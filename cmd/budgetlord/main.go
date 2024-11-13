package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tanq16/budgetlord/internal/api"
	"github.com/tanq16/budgetlord/internal/config"
	"github.com/tanq16/budgetlord/internal/storage/jsonfile"
	"github.com/tanq16/budgetlord/internal/web"
)

var categories = []string{
	"Food",
	"Transportation",
	"Housing",
	"Utilities",
	"Entertainment",
	"Healthcare",
	"Shopping",
	"Other",
}

func runServer() {
	cfg := config.NewConfig()

	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	storage, err := jsonfile.New(dataDir + "/expenses.json")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	handler := api.NewHandler(storage)

	http.HandleFunc("/expense", handler.AddExpense)
	http.HandleFunc("/expenses", handler.GetExpenses)
	http.HandleFunc("/table", handler.ServeTableView)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if err := web.ServeTemplate(w, "index.html"); err != nil {
			http.Error(w, "Failed to serve template", http.StatusInternalServerError)
			return
		}
	})

	log.Printf("Starting server on port %s...", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func readNonEmptyInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			return input
		}
		fmt.Println("This field is required. Please try again.")
	}
}

func getCategory() string {
	fmt.Println("\nAvailable categories:")
	for i, category := range categories {
		fmt.Printf("%d. %s\n", i+1, category)
	}

	for {
		fmt.Print("\nSelect category (1-8): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(categories) {
			return categories[num-1]
		}
		fmt.Println("Invalid selection. Please try again.")
	}
}

func getAmount() float64 {
	for {
		input := readNonEmptyInput("Enter amount: ")
		amount, err := strconv.ParseFloat(input, 64)
		if err == nil && amount > 0 {
			return amount
		}
		fmt.Println("Invalid amount. Please enter a positive number.")
	}
}

func getDate() time.Time {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter date (YYYY-MM-DD, press Enter for today): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return time.Now()
	}

	// Parse the input date and set it to 2 PM in the current timezone
	if date, err := time.ParseInLocation("2006-01-02", input, time.Local); err == nil {
		return time.Date(
			date.Year(), date.Month(), date.Day(),
			14, 0, 0, 0,
			time.Local,
		)
	}

	fmt.Println("Invalid date format, using current time.")
	return time.Now()
}

func runClient(serverAddr string) {
	name := readNonEmptyInput("Enter expense name: ")
	category := getCategory()
	amount := getAmount()
	date := getDate()

	expense := api.ExpenseRequest{
		Name:     name,
		Category: category,
		Amount:   amount,
		Date:     date,
	}

	jsonData, err := json.Marshal(expense)
	if err != nil {
		log.Fatalf("Failed to marshal expense data: %v", err)
	}

	url := fmt.Sprintf("http://%s/expense", serverAddr)
	resp, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	resp.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(resp)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatalf("Server returned error: %s", response.Status)
	}

	fmt.Println("Expense added successfully!")
}

func main() {
	isServer := flag.Bool("serve", true, "Run as server (default true)")
	isClient := flag.Bool("client", false, "Run as client")
	serverAddr := flag.String("addr", "localhost:8080", "Server address (for client mode)")

	flag.Parse()

	// If both flags are provided, prefer client mode
	if *isClient {
		runClient(*serverAddr)
	} else if *isServer {
		runServer()
	}
}