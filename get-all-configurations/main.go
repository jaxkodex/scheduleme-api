package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Configuration struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type User struct {
	UserID    string   `json:"userid"`
	Username  string   `json:"username"`
	FirstName string   `json:"firstName,omitempty"`
	LastName  string   `json:"lastName,omitempty"`
	Email     string   `json:"email,omitempty"`
	Phone     string   `json:"phone,omitempty"`
	Status    []string `json:"status"`
}

type AllConfigurationsResponse struct {
	User           User            `json:"user"`
	Configurations []Configuration `json:"configurations"`
}

func getUserIdFromAuthHeader(authHeader string) string {
	// Example: Bearer <token> (token is userId for this example)
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received request: %+v", request)

	authHeader := request.Headers["Authorization"]
	userId := getUserIdFromAuthHeader(authHeader)
	if userId == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 403,
			Body:       `{"message": "Forbidden: No user id in token"}`,
		}, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"message": "Internal server error"}`,
		}, nil
	}

	svc := dynamodb.NewFromConfig(cfg)

	// Get user from SCHEDULEME_USERS
	userResult, err := svc.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("SCHEDULEME_USERS"),
		Key: map[string]types.AttributeValue{
			"userid": &types.AttributeValueMemberS{Value: userId},
		},
	})
	if err != nil || userResult.Item == nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       `{"message": "User not found"}`,
		}, nil
	}

	user := User{
		UserID:   userResult.Item["userid"].(*types.AttributeValueMemberS).Value,
		Username: userResult.Item["username"].(*types.AttributeValueMemberS).Value,
		Status:   []string{userResult.Item["status"].(*types.AttributeValueMemberS).Value},
	}
	if firstName, ok := userResult.Item["firstName"]; ok {
		user.FirstName = firstName.(*types.AttributeValueMemberS).Value
	}
	if lastName, ok := userResult.Item["lastName"]; ok {
		user.LastName = lastName.(*types.AttributeValueMemberS).Value
	}
	if email, ok := userResult.Item["email"]; ok {
		user.Email = email.(*types.AttributeValueMemberS).Value
	}
	if phone, ok := userResult.Item["phone"]; ok {
		user.Phone = phone.(*types.AttributeValueMemberS).Value
	}

	// Query USER_CONFIGURATIONS for this user
	configResult, err := svc.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String("USER_CONFIGURATIONS"),
		KeyConditionExpression: aws.String("userid = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userId},
		},
	})
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"message": "Failed to get configurations"}`,
		}, nil
	}

	configs := []Configuration{}
	for _, item := range configResult.Items {
		config := Configuration{
			ID:    item["id"].(*types.AttributeValueMemberS).Value,
			Name:  item["name"].(*types.AttributeValueMemberS).Value,
			Value: item["value"].(*types.AttributeValueMemberS).Value,
		}
		configs = append(configs, config)
	}

	resp := AllConfigurationsResponse{
		User:           user,
		Configurations: configs,
	}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"message": "Internal server error"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(respJSON),
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
