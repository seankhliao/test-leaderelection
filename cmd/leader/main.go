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
	// setup client to talk to api server using service account credentials
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

	// get a unique identity for ourselves
	hostname, err := os.Hostname()
	if err != nil {
		klog.ErrorS(err, "get hostname")
		os.Exit(1)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// runs in a leader election loop
	// panics on failure
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		// this is the lease we create/update if we win the leader
		Lock: &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "test-lease",
			},
			Client: client.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: hostname,
			},
		},
		// recommended defaults
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		// TODO, ensure exit from critical work before canceling context
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			// main work should happen here
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
				// TODO: ensure work loop exit before canceling leaderelection ctx
				cancel()
				klog.InfoS("stopped leading")
			},
			OnNewLeader: func(identity string) {
				// just notifications
				klog.InfoS("new leader", "id", identity)
			},
		},
	})
}
