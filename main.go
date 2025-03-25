package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type cliCommand struct {
	name        string
	description string
	callback    func() error
}

type config struct {
	Next string
	Prev string
}
type LocationAreaResponse struct {
	Count    int            `json:"count"`
	Next     *string        `json:"next"`     // Pointer because it could be null
	Previous *string        `json:"previous"` // Pointer because it could be null
	Results  []LocationArea `json:"results"`
}

type LocationArea struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func main() {

	var command map[string]cliCommand
	config := &config{}

	command = map[string]cliCommand{
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    func() error { return commandHelp(command) },
		},
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"map": {
			name:        "map",
			description: "Displays the names of 20 location areas",
			callback:    func() error { return commandMap(config) },
		},
		"mapb": {
			name:        "mapb",
			description: "Displays the names of Previous 20 location areas",
			callback:    func() error { return commandMapb(config) },
		},
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("Pokedex > ")
		scanner.Scan()
		input := cleanInput(scanner.Text())
		if len(input) == 0 {
			continue
		}
		// 	res := cleanInput(scanner.Text())
		// 	fmt.Println("Your command was: " + res[0])
		cmd, ok := command[input[0]]
		if ok {
			err := cmd.callback()
			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println("Unknown command")
		}
	}

}

func commandExit() error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandMap(config *config) error {

	url := "https://pokeapi.co/api/v2/location-area"
	if config.Next != "" {
		url = config.Next
	}
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
	}
	if err != nil {
		return err
	}

	response := LocationAreaResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Next != nil {
		config.Next = *response.Next
	} else {
		config.Next = ""
	}

	if response.Previous != nil {
		config.Prev = *response.Previous
	} else {
		config.Prev = ""
	}
	for _, area := range response.Results {
		fmt.Println(area.Name)
	}

	return nil

}

func commandMapb(config *config) error {

	// If Previous is empty, we're on the first page
	if config.Prev == "" {
		fmt.Println("You're on the first page")
		return nil
	}

	res, err := http.Get(config.Prev)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode > 299 {
		return fmt.Errorf("response failed with status code: %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	response := LocationAreaResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	// Update config with new Next and Previous URLs
	if response.Next != nil {
		config.Next = *response.Next
	} else {
		config.Next = ""
	}

	if response.Previous != nil {
		config.Prev = *response.Previous
	} else {
		config.Prev = ""
	}

	// Print just the names
	for _, area := range response.Results {
		fmt.Println(area.Name)
	}

	return nil
}

func commandHelp(commands map[string]cliCommand) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()

	orderedCmds := []string{"help", "map", "mapb", "exit"}

	for _, cmd := range orderedCmds {
		if cmd, ok := commands[cmd]; ok {
			fmt.Printf("%s: %s\n", cmd.name, cmd.description)
		}
	}
	return nil

}

func cleanInput(text string) []string {
	e := strings.Fields(strings.ToLower(text))
	return e

}
