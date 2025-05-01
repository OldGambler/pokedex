package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/OldGambler/pokedex/internal/pokecache"
)

type cliCommand struct {
	name        string
	description string
	callback    func(con *Config, args ...string) error
}

type Pokemon struct {
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
}

var commands map[string]cliCommand

func main() {
	cache := pokecache.NewCache(5 * time.Minute)
	con := &Config{
		Cache:   cache,
		Pokedex: make(map[string]Pokemon),
	}

	commands = map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    commandHelp,
		},
		"map": {
			name:        "map",
			description: "Displays locations",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Displays previous locations",
			callback:    commandMapback,
		},
		"explore": {
			name:        "explore",
			description: "Lists pokemon in location",
			callback:    commandExplore,
		},
		"catch": {
			name:        "catch",
			description: "throws a pokeball, to hopefully catch a pokemon",
			callback:    commandCatch,
		},
		"inspect": {
			name:        "inspect",
			description: "looks at information of selected pokemon",
			callback:    commandInspect,
		},
	}

	userInput := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		userInput.Scan()
		text := userInput.Text()
		cleanText := strings.ToLower(strings.TrimSpace(text))
		words := strings.Fields(cleanText)
		if len(words) > 0 {
			command := words[0]
			args := words[1:]
			if cmd, exists := commands[command]; exists {
				err := cmd.callback(con, args...)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				fmt.Println("unknown command")
			}
		}
	}

}

type Config struct {
	Next     string
	Previous string
	Cache    *pokecache.Cache
	Pokedex  map[string]Pokemon
}

func commandExit(con *Config, args ...string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(con *Config, args ...string) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()
	for _, cmd := range commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

type LocationAreaResponse struct {
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

func commandMap(con *Config, args ...string) error {
	url := "https://pokeapi.co/api/v2/location-area"
	if con.Next != "" {
		url = con.Next
	}

	var body []byte
	var err error

	if cachedData, found := con.Cache.Get(url); found {
		fmt.Println("using cached data for location areas")
		body = cachedData
	} else {
		fmt.Println("fetching fresh data for location areas")
		res, err := http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		con.Cache.Add(url, body)
	}
	var response LocationAreaResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	con.Next = response.Next
	con.Previous = response.Previous

	for _, item := range response.Results {
		fmt.Println(item.Name)
	}

	return nil
}

func commandMapback(con *Config, args ...string) error {
	if con.Previous == "" {
		fmt.Println("you're on the first page")
		return nil
	}
	url := "https://pokeapi.co/api/v2/location-area"
	if con.Previous != "" {
		url = con.Previous
	}

	var body []byte
	var err error

	if cachedData, found := con.Cache.Get(url); found {
		fmt.Println("using cached data for location areas")
		body = cachedData
	} else {
		fmt.Println("fetching fresh data for location areas")
		res, err := http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		con.Cache.Add(url, body)
	}

	var response LocationAreaResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	con.Next = response.Next
	con.Previous = response.Previous

	for _, item := range response.Results {
		fmt.Println(item.Name)
	}

	return nil

}

func commandExplore(c *Config, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("the explore command requires exactly one argument: the location area name")
	}
	locationAreaName := args[0]
	return explore(c, locationAreaName)
}

func explore(con *Config, lan string) error {
	url := "https://pokeapi.co/api/v2/location-area/" + lan
	if lan == "" {
		return fmt.Errorf("Must provide a location area name")
	}

	var body []byte
	var err error

	if cachedData, found := con.Cache.Get(url); found {
		fmt.Println("using cached data for pokemon location")
		body = cachedData
	} else {
		fmt.Println("fetching fresh data for pokemon location")
		res, err := http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		con.Cache.Add(url, body)
	}

	type LocationAreaDetail struct {
		PokemonEncounters []struct {
			Pokemon struct {
				Name string `json:"name"`
			} `json:"pokemon"`
		} `json:"pokemon_encounters"`
	}

	var response LocationAreaDetail
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	fmt.Printf("Exploring %s...\n", lan)
	fmt.Println("Found Pokemon:")
	for _, encounter := range response.PokemonEncounters {
		fmt.Printf(" - %s\n", encounter.Pokemon.Name)
	}
	return nil
}

func commandCatch(con *Config, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("The catch command requires the name of the pokemon")
	}
	pokemon := args[0]
	return catch(con, pokemon)
}

func catch(con *Config, pokemon string) error {
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemon)
	url := "https://pokeapi.co/api/v2/pokemon/" + pokemon
	randomNumber := rand.Intn(100)

	var body []byte
	var err error

	if cachedData, found := con.Cache.Get(url); found {
		fmt.Println("using cached data for pokemon data")
		body = cachedData
	} else {
		fmt.Println("fetching fresh data for pokemon data")
		res, err := http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		body, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		con.Cache.Add(url, body)
	}

	var poke Pokemon
	err = json.Unmarshal(body, &poke)
	if err != nil {
		return err
	}
	if randomNumber > poke.BaseExperience {
		fmt.Println(pokemon, "caught!")
		con.Pokedex[poke.Name] = poke
	} else {
		fmt.Println(pokemon, "escaped!")
	}
	return nil
}

func commandInspect(con *Config, args ...string) error {
	if len(args) != 1 {
		return errors.New("you must provide a pokemon name")
	}

	name := args[0]
	pokemon, ok := con.Pokedex[name]
	if !ok {
		return errors.New("you have not caught that pokemon")
	}

	fmt.Println("Name:", pokemon.Name)
	fmt.Println("Height:", pokemon.Height)
	fmt.Println("Weight:", pokemon.Weight)
	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf("  -%s: %v\n", stat.Stat.Name, stat.BaseStat)
	}
	fmt.Println("Types:")
	for _, typeInfo := range pokemon.Types {
		fmt.Println("  -", typeInfo.Type.Name)
	}
	return nil
}

func cleanInput(text string) []string {
	if len(text) == 0 {
		return []string{}
	}
	trimmed := strings.TrimSpace(text)
	words := strings.Fields(trimmed)
	for i, word := range words {
		words[i] = strings.ToLower(word)
	}
	return words
}
