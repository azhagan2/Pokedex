package main

import (
	"azhagan2/internal/pokecache"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var baseURL = "https://pokeapi.co/api/v2/location-area"
var pokemonURL = "https://pokeapi.co/api/v2/pokemon"

type BaseEncounter struct {
	BaseExperience int `json:"base_experience"`
}

type PokemonEncountersResponse struct {
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

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

	cache := pokecache.NewCache(10 * time.Minute)

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
			callback:    func() error { return commandMap(config, cache) },
		},
		"mapb": {
			name:        "mapb",
			description: "Displays the names of Previous 20 location areas",
			callback:    func() error { return commandMapb(config, cache) },
		},
		"explore": {
			name:        "explore {location-area}",
			description: "Displays the list of Pokemon located in a location area",
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

		if len(input) > 1 && strings.ToLower(input[0]) == "explore" {
			Explore(input[1], cache)
		} else if len(input) > 1 && strings.ToLower(input[0]) == "catch" {
			Catch(input[1], cache)
		} else {
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

}

func commandExit() error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandMap(config *config, cache *pokecache.Cache) error {

	if config.Next != "" {
		baseURL = config.Next
	}

	if cachedData, found := cache.Get(baseURL); found {
		fmt.Println("Cache hit! Using cached data...")
		return processLocationAreaResponse(cachedData, config)
	}

	// If not in cache, fetch the data from the API
	fmt.Println("Cache miss! Fetching from API...")

	res, err := http.Get(baseURL)
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

	cache.Add(baseURL, body)

	// Step 6: Process the data using the helper
	return processLocationAreaResponse(body, config)

}

func processLocationAreaResponse(data []byte, config *config) error {
	respone := LocationAreaResponse{}
	err := json.Unmarshal(data, &respone)
	if err != nil {
		return fmt.Errorf("failed to Unmarshal data: %w", err)
	}

	if respone.Next != nil {
		config.Next = *respone.Next
	} else {
		config.Next = ""
	}

	if respone.Previous != nil {
		config.Prev = *respone.Previous
	} else {
		config.Prev = ""
	}

	fmt.Println("Location Areas: ")
	for _, area := range respone.Results {
		fmt.Println(" -", area.Name)
	}

	return nil
}

func commandMapb(config *config, cache *pokecache.Cache) error {

	// If Previous is empty, we're on the first page
	if config.Prev == "" {
		fmt.Println("You're on the first page")
		return nil
	}

	baseURL := config.Prev

	if cachedData, found := cache.Get(baseURL); found {
		fmt.Println("Cache hit! Using cached data...")
		return processLocationAreaResponse(cachedData, config)
	}

	// If not in cache, fetch the data from the API
	fmt.Println("Cache miss! Fetching from API...")

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

	cache.Add(baseURL, body)

	// Step 6: Process the data using the helper
	return processLocationAreaResponse(body, config)
}

func commandHelp(commands map[string]cliCommand) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()

	orderedCmds := []string{"explore", "help", "map", "mapb", "exit"}

	for _, cmd := range orderedCmds {
		if cmd, ok := commands[cmd]; ok {
			fmt.Printf("%s: %s\n", cmd.name, cmd.description)
		}
	}
	return nil

}

func Explore(a string, cache *pokecache.Cache) error {
	// fmt.Println("Fuck. I can't provide") // Just checking ah
	poke_name_url := pokemonURL + "/" + a
	// fmt.Println(poke_url)

	if cachedData, found := cache.Get(poke_name_url); found {
		fmt.Println("Cache hit! Using cached data...")
		return get_pokemon_name_from_location_area(cachedData)
	}

	fmt.Println("Cache miss! Fetching from API...")

	res, err := http.Get(poke_name_url)
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

	// response := PokemonEncountersResponse{}
	// err = json.Unmarshal(body, &response)
	// if err != nil {
	// 	return err
	// }
	cache.Add(poke_name_url, body)

	// Step 6: Process the data using the helper
	return get_pokemon_name_from_location_area(body)
}

func Catch(a string, cache *pokecache.Cache) error {
	poke_url := pokemonURL + "/" + a
	// fmt.Println(poke_url)

	if cachedData, found := cache.Get(poke_url); found {
		fmt.Println("Cache hit! Using cached data...")
		return get_base_encounter(cachedData)
	}
	fmt.Println(poke_url)
	fmt.Println("Cache miss! Catching from API...")

	res, err := http.Get(poke_url)
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

	// response := PokemonEncountersResponse{}
	// err = json.Unmarshal(body, &response)
	// if err != nil {
	// 	return err
	// }
	cache.Add(poke_url, body)

	// Step 6: Process the data using the helper
	return get_base_encounter(body)
}

func get_pokemon_name_from_location_area(data []byte) error {
	response := PokemonEncountersResponse{}
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	for _, items := range response.PokemonEncounters {
		fmt.Println(items.Pokemon.Name)
	}
	return nil
}

func get_base_encounter(data []byte) error {
	response := BaseEncounter{}
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	fmt.Println(response.BaseExperience)

	rand.Seed(time.Now().UnixNano())

	n := response.BaseExperience
	success := rand.Intn(n) == 0

	fmt.Println("Throwing a Pokeball at pikachu...")

	if success {
		fmt.Println("pikachu was caught!")
	} else {
		fmt.Println("Pikachu escaped!")
	}

	return nil
}

func cleanInput(text string) []string {
	e := strings.Fields(strings.ToLower(text))
	return e

}
