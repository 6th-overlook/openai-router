package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

type UserReq struct {
	GptMessage string `json:"gpt_message"`
}

// getParameterFromSSM retrieves a parameter from AWS Systems Manager Parameter Store.
// It takes a parameter name as input and returns the parameter value as a string.
func getParameterFromSSM(parameterName string) (string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})

	if err != nil {
		return "", err
	}

	ssmSvc := ssm.New(sess)
	input := &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(true),
	}

	res, err := ssmSvc.GetParameter(input)
	if err != nil {
		return "", err
	}

	return *res.Parameter.Value, nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var gptRequest UserReq
	err := json.Unmarshal([]byte(request.Body), gptRequest)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Error in parsing JSON",
			StatusCode: 400,
		}, nil
	}

	apiKey, err := getParameterFromSSM("openai-api-key")

	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	openAIRequest := OpenAIRequest{
		Model: "gpt-4",
		Messages: []OpenAIMessage{
			{Role: "system", Content: "You are a conversational companion intended converse actively with the user, asking questions and providing appropriate reponses to statements and questions"},
			{Role: "user", Content: gptRequest.GptMessage}},
	}

	openAIReqJSON, err := json.Marshal(openAIRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Unable to marshal the JSON body",
			StatusCode: 500,
		}, nil
	}
	buffer := bytes.NewBuffer(openAIReqJSON)

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", buffer)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Unable to create the HTTP request",
			StatusCode: 500,
		}, nil
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Unable to send the HTTP request",
			StatusCode: 500,
		}, nil
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Unable to read the response body",
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(body),
		StatusCode: 200,
	}, nil
}

// main is the entry point of the application. It starts the AWS Lambda function with the handler.
func main() {
	lambda.Start(handler)
}
