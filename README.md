# PC Part Picker Scraper

This project is a scraper for the PC Part Picker website.

## Project structure

The project consists of the following significant modules:

1. `main.go`: The entry point to the program.
2. `pkg/scraper/scraper.go`: The main module that handles the web scraping process.
3. `internal/models/parts.go` y `price.go`: These modules contain the definitions of the data models used.
4. `internal/utils/utils.go`: Contains utility/help functions used throughout the project.
5. `go.mod`: The Go module file that manages the project dependencies.

## How to use

The program can be run using the `go run main.go` command from the root of the project.

Make sure you have all dependencies installed as specified in `go.mod`.