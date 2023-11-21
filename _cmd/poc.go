package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func repr(s thermState) string {
	return fmt.Sprintf("1: rel1=%t\tsens01=%t\tsens11=%t\t\n2: rel2=%t\tsens02=%t\tsens12=%t\t\n3: rel3=%t\tsens03=%t\tsens13=%t\t\n",
		s.Rel1, s.Sens01, s.Sens11, s.Rel2, s.Sens02, s.Sens12, s.Rel3, s.Sens03, s.Sens13)
}

func main() {
	log.Println("Starting")
	var therm *therm

	cb := func() {
		log.Println("Interrupt!")
		log.Println("\n" + repr(therm.GetState()))
	}

	therm, err := NewTherm(cb)
	if err != nil {
		log.Fatal(err)
	}
	defer therm.Close()
	log.Println("Therm initialized")

	// Main loop
	t := 500 * time.Millisecond
	i := 0
	go func() {
		for {
			therm.SetState(thermState{
				Rel1: (i & 0x1) > 0,
				Rel2: (i & 0x2) > 0,
				Rel3: (i & 0x4) > 0,
			})
			i += 1
			time.Sleep(t)
		}
	}()
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT)
	<-sigint
	log.Println("Recived interrupt, shutting down")
}
