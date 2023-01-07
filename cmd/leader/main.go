package main

import (
	"context"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
)

func main() {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		klog.ErrorS(err, "get rest config")
		os.Exit(1)
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.ErrorS(err, "get rest client")
		os.Exit(1)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-lease",
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: os.Getenv("POD_NAME"),
		},
	}

	// panics on die
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				dur := 5 * time.Second
				klog.InfoS("leading", "tick_dur", dur)
				tick := time.NewTicker(dur)
				defer tick.Stop()
			leadLoop:
				for {
					select {
					case <-tick.C:
						klog.InfoS("still leading")
					case <-ctx.Done():
						klog.InfoS("leading cancelled")
						break leadLoop
					}
				}
			},
			OnStoppedLeading: func() {
				cancel()
				klog.InfoS("stopped leading")
			},
			OnNewLeader: func(identity string) {
				klog.InfoS("new leader", "id", identity)
			},
		},
	})
}
