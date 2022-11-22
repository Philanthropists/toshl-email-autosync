package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/store/nosql/dynamodb"
)

func main() {
	table := flag.String("table", "toshl-data", "DynamoDB table to read from")
	flag.Parse()

	ctx := context.Background()
	client, err := dynamodb.NewDynamoDBClient(ctx, "us-east-1")
	if err != nil {
		panic(err)
	}

	res, err := client.Scan(ctx, *table)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Table: %s\n", *table)
	fmt.Printf("Scan: %+v\n", res)

	// ---

	res2, err := client.GetItem(ctx, *table, map[string]interface{}{
		"Id": 1,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Get Item: 1\n")
	fmt.Printf("Item: %+v\n", res2)

	// ---
	const field = "LastProcessedDate"
	lastProcDate, ok := res2[field]
	if !ok {
		log.Fatalf("there is no date in item [%v]", res2)
	}

	exp := fmt.Sprintf("set %s = :r", field)
	value := time.Now().Format(time.RFC822Z)
	fmt.Printf("updating item 1, with exp [%s] and :r = \"%s\"\n", exp, value)

	err = client.UpdateItem(ctx, *table, map[string]interface{}{
		"Id": 1,
	}, map[string]interface{}{
		":r": value,
	}, exp)

	if err != nil {
		panic(err)
	}

	// ---
	res2, err = client.GetItem(ctx, *table, map[string]interface{}{
		"Id": 1,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Get Item: 1\n")
	fmt.Printf("Item: %+v\n", res2)

	// ---
	fmt.Println("\nreturning to original values")
	err = client.UpdateItem(ctx, *table, map[string]interface{}{
		"Id": 1,
	}, map[string]interface{}{
		":r": lastProcDate,
	}, exp)

	if err != nil {
		panic(err)
	}

	// ---
	fmt.Println("\nmaking sure it was restored")
	res2, err = client.GetItem(ctx, *table, map[string]interface{}{
		"Id": 1,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Get Item: 1\n")
	fmt.Printf("Item: %+v\n", res2)

}
