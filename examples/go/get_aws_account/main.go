package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func main() {
	fmt.Println("=== Getting AWS Account Information ===")

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create STS client
	stsClient := sts.NewFromConfig(cfg)

	// Get caller identity
	result, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatalf("Failed to get caller identity: %v", err)
	}

	fmt.Printf("Account ID: %s\n", *result.Account)
	fmt.Printf("User ID: %s\n", *result.UserId)
	fmt.Printf("ARN: %s\n", *result.Arn)

	fmt.Println("\n=== Copy the Account ID above for use in the deletion test ===")
}
