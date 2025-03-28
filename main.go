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

type Pokemon_details struct {
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	Stats          []struct {
		BaseStat int `json:"base_stat"`
		Effort   int `json:"effort"`
		Stat     struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Slot int `json:"slot"`
		Type struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"type"`
	} `json:"types"`
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

type game struct {
	pokedex map[string]Pokemon
}

type Pokemon struct {
	Name           string
	BaseExperience int
	Inspection     Pokemon_details
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

var g = &game{
	pokedex: make(map[string]Pokemon),
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
		"catch": {
			name:        "catch {pokemon}",
			description: "catches the pokemon into the pokedex, user is giving",
		},
		"inspect": {
			name:        "inspect {pokemon}",
			description: "inspects the caught pokemon",
		},
		"pokedex": {
			name:        "pokedex",
			description: "shows all the pokemon caught",
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
		} else if len(input) > 1 && strings.ToLower(input[0]) == "inspect" {
			g.inspect(input[1])
		} else if strings.ToLower(input[0]) == "pokedex" {
			g.commandPokedex()
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

	orderedCmds := []string{"catch", "explore", "help", "inspect", "map", "mapb", "pokdex", "exit"}

	for _, cmd := range orderedCmds {
		if cmd, ok := commands[cmd]; ok {
			fmt.Printf("%s: %s\n", cmd.name, cmd.description)
		}
	}
	return nil

}

func Explore(a string, cache *pokecache.Cache) error {
	// fmt.Println("Fuck. I can't provide") // Just checking ah
	poke_name_url := baseURL + "/" + a
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
		return g.get_base_encounter(cachedData, a)
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
	return g.get_base_encounter(body, a)
}

func get_pokemon_name_from_location_area(data []byte) error {

	fmt.Println("Inside the get_pokemon_name_frm_location_area")
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

func (g *game) get_base_encounter(data []byte, a string) error {
	response := Pokemon_details{}
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	// fmt.Println(response.BaseExperience)

	rand.Seed(time.Now().UnixNano())

	n := max(response.BaseExperience/20, 1)
	success := rand.Intn(n) == 0

	fmt.Printf("Throwing a Pokeball at %v...\n", a)

	if success {
		fmt.Printf("%v was caught!\n", a)

		g.pokedex[a] = Pokemon{
			Name:           a,
			BaseExperience: response.BaseExperience,
			Inspection: Pokemon_details{
				Height: response.Height,
				Weight: response.Weight,
				Stats:  response.Stats,
				Types:  response.Types,
			},
		}

	} else {
		fmt.Printf("%v escaped!\n", a)
	}

	fmt.Println("Pokedex now contains:", len(g.pokedex), "Pokemon")
	for name := range g.pokedex {
		fmt.Println("- " + name)
	}

	return nil
}

func (g *game) inspect(a string) {

	fmt.Println("Looking for Pokémon:", a)
	fmt.Println("Pokémon in pokedex:")
	for key := range g.pokedex {
		fmt.Println("- '" + key + "'")
	}

	pokemon, found := g.pokedex[a]
	if !found {
		fmt.Printf("you have not caught that pokemon: %v\n", a)
		return
	} else {
		fmt.Printf("Name: %v\n", pokemon.Name)
		fmt.Printf("Height: %v\n", pokemon.Inspection.Height)
		fmt.Printf("Weight: %v\n", pokemon.Inspection.Weight)

		fmt.Println("Stats:")

		for _, stat := range pokemon.Inspection.Stats {
			fmt.Printf(" -%v: %v\n", stat.Stat.Name, stat.BaseStat)
		}
		fmt.Println("Types:")
		for _, stat := range pokemon.Inspection.Types {
			fmt.Printf(" -%v\n", stat.Type.Name)
		}
	}

}

func (g *game) commandPokedex() {
	for key := range g.pokedex {
		fmt.Println("- '" + key + "'")
	}
}

func cleanInput(text string) []string {
	e := strings.Fields(strings.ToLower(text))
	return e

}
