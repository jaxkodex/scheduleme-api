package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type User struct {
	UserID    string   `json:"userid"`
	Username  string   `json:"username"`
	FirstName string   `json:"firstName,omitempty"`
	LastName  string   `json:"lastName,omitempty"`
	Email     string   `json:"email,omitempty"`
	Phone     string   `json:"phone,omitempty"`
	Status    []string `json:"status"`
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get userId from path parameters
	userId := request.PathParameters["userId"]
	if userId == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"message": "userId is required"}`,
		}, nil
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"message": "Internal server error"}`,
		}, nil
	}

	// Create DynamoDB client
	svc := dynamodb.NewFromConfig(cfg)

	// Get item from DynamoDB
	result, err := svc.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("SCHEDULEME_USERS"),
		Key: map[string]types.AttributeValue{
			"userid": &types.AttributeValueMemberS{Value: userId},
		},
	})
	if err != nil {
		log.Printf("Failed to get item from DynamoDB: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"message": "Internal server error"}`,
		}, nil
	}

	// Check if item exists
	if result.Item == nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       `{"message": "User not found"}`,
		}, nil
	}

	// Convert DynamoDB item to User struct
	user := User{
		UserID:   result.Item["userid"].(*types.AttributeValueMemberS).Value,
		Username: result.Item["username"].(*types.AttributeValueMemberS).Value,
		Status:   []string{result.Item["status"].(*types.AttributeValueMemberS).Value},
	}

	// Add optional fields if they exist
	if firstName, ok := result.Item["firstName"]; ok {
		user.FirstName = firstName.(*types.AttributeValueMemberS).Value
	}
	if lastName, ok := result.Item["lastName"]; ok {
		user.LastName = lastName.(*types.AttributeValueMemberS).Value
	}
	if email, ok := result.Item["email"]; ok {
		user.Email = email.(*types.AttributeValueMemberS).Value
	}
	if phone, ok := result.Item["phone"]; ok {
		user.Phone = phone.(*types.AttributeValueMemberS).Value
	}

	// Convert user to JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to marshal user: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"message": "Internal server error"}`,
		}, nil
	}

	// Return successful response
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(userJSON),
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
