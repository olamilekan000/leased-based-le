package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func main() {
	var (
		leaseLockName      string
	)

	const leaseLockNamespace = "default"

	identity := os.Getenv("POD_NAME")

	flag.StringVar(&leaseLockName, "lease-name", "", "Name of lease lock")
	flag.Parse()

	if leaseLockName == "" {
		log.Fatal("missing lease-name flag")
	}

	// Create a context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("Redis is ready. Starting the application.")

	rcl := NewRedisClient()
	_, err := rcl.redisCL.Ping(ctx).Result()
	if err != nil {
		fmt.Println("err: pingingngn----", err.Error())
	}

	router := gin.Default()
	port := ":8881"

	AddRoutes(router)

	srv := &http.Server{
		Addr:    port,
		Handler: router,
	}

	go func() {
		fmt.Println(fmt.Sprintf("Starting server at %s", port))
		if err := srv.ListenAndServe(); err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
	}()

	// Load the Kubernetes configuration
	config, err := rest.InClusterConfig()
	if err != nil {

		log.Fatal("err: pingingngn----", err.Error())
		return
	}

	// Create a Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	lock, err := resourcelock.New(
		resourcelock.LeasesResourceLock,
		leaseLockNamespace,
		leaseLockName,
		clientset.CoreV1(),
		clientset.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: identity,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	leConf := leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 10 * time.Second,
		RenewDeadline: 5 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				for {
					fmt.Println("running", identity)

					var rVal *string
					var num int

					rVal, err = rcl.Get(ctx, rKey)
					if err != nil {

						fmt.Println("err:", err)
					}

					fmt.Println("AAAAAAAA", rVal)

					if rVal != nil {
						num, err = strconv.Atoi(*rVal)
						if err != nil {
							fmt.Println("Error: convert", err)
							return
						}
					}

					num += 1

					err := rcl.Set(ctx, rKey, strconv.Itoa(num))
					if err != nil {
						log.Fatal("err: redis----", err.Error())

						return
					}

					time.Sleep(5 * time.Second)
				}
			},
			OnStoppedLeading: func() {
				fmt.Println(fmt.Sprintf("Pod %s stopped leading", identity))
			},
			OnNewLeader: func(id string) {
				if id == identity {
					fmt.Println(fmt.Sprintf("%s Pod still the leader", id))

					return
				}

				fmt.Println(fmt.Sprintf("new Pod elected: %s", id))
			},
		},
	}

	// Start leader election loop.
	elector, err := leaderelection.NewLeaderElector(leConf)
	if err != nil {
		log.Fatal(err)
	}

	elector.Run(ctx)

	// Wait for termination signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh

	// Cancel the context and exit
	cancel()
}
