package main

import (
	"fmt"
	"log"
	"strings"
)

func repr(s thermState) string {
	return fmt.Sprintf("1: rel1=%t\tsens01=%t\tsens11=%t\t\n2: rel2=%t\tsens02=%t\tsens12=%t\t\n3: rel3=%t\tsens03=%t\tsens13=%t\t\n",
		s.Rel1, s.Sens01, s.Sens11, s.Rel2, s.Sens02, s.Sens12, s.Rel3, s.Sens03, s.Sens13)
}

const usage string = "USAGE:\n  Reln: Toggle relay #n\n  State: Display status\n  Exit: Exit"

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

	// CLI loop
	fmt.Println(usage)
	var cmd string
	for loop := true; loop; {
		cmd = ""
		fmt.Print(">")
		n, err := fmt.Scanln(&cmd)
		if err != nil {
			fmt.Println("ERROR: unable to parse command", err)
			fmt.Println(usage)
			continue
		}
		if n != 1 {
			fmt.Println("ERROR: too many arguments")
			fmt.Println(usage)
			continue
		}
		s := therm.GetState()
		switch strings.ToUpper(cmd) {
		case "REL1":
			s.Rel1 = !s.Rel1
			therm.SetState(s)
		case "REL2":
			s.Rel2 = !s.Rel2
			therm.SetState(s)
		case "REL3":
			s.Rel3 = !s.Rel3
			therm.SetState(s)
		case "STATE":
			fmt.Println("Current state:")
			fmt.Println(repr(s))
		case "EXIT":
			fmt.Println("Bye!")
			loop = false
			break
		default:
			fmt.Println(usage)
		}
	}
}
