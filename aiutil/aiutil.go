package aiutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/invopop/jsonschema"

	"github.com/teejays/gokutil/env/envutil"
	jsonutil "github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/log"
)

type Client struct {
	httpClient *http.Client
	apiKey     string
}

// Shared Types
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	Refusal string `json:"refusal,omitempty"` // Optional: Only in responses, if the assistant refuses to respond.
}

type Model string

const (
	Model_GPT_4o          Model = "gpt-4o"            // Doesn't support response type "json_schema"
	Model_GPT_4o_20240806 Model = "gpt-4o-2024-08-06" // 30k TMP
	Model_GPT_4o_mini     Model = "gpt-4o-mini"       // 200k limit
	Model_GPT_3_5_turbo   Model = "gpt-3.5-turbo"     // 200k limit
)

type Role string

const (
	Role_Assistant Role = "assistant"
	Role_User      Role = "user"
	Role_System    Role = "system" // Not used in Assistants V2 API
)

// Request Types
type (
	ChatRequest struct {
		Model            Model                  `json:"model"`
		Messages         []Message              `json:"messages"`
		ResponseFormat   *MessageResponseFormat `json:"response_format,omitempty"`
		FrequencyPenalty float64                `json:"frequency_penalty,omitempty"` // -2 to 2: Low value = more deterministic, high value = more random
		Temperature      float64                `json:"temperature,omitempty"`       // 0-2: Low value = more deterministic, high value = more random
	}

	MessageResponseFormat struct {
		Type       string                          `json:"type,omitempty"`
		JSONSchema MessageResponseFormatJSONSchema `json:"json_schema,omitempty"`
	}

	MessageResponseFormatJSONSchema struct {
		Description string     `json:"description,omitempty"`
		Name        string     `json:"name,omitempty"`
		Strict      bool       `json:"strict,omitempty"`
		Schema      JSONSchema `json:"schema,omitempty"`
	}

	JSONSchema []byte
)

func (v JSONSchema) MarshalJSON() ([]byte, error) {
	return v, nil
}

func (v *JSONSchema) UnmarshalJSON(data []byte) error {
	*v = data
	return nil
}

func NewJSONSchemaFromStruct(v interface{}) (JSONSchema, error) {
	js := jsonschema.Reflect(v)
	js.Type = "object"
	b, err := js.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return JSONSchema(b), nil
}

// Response Types
type (
	ChatDefaultResponse struct {
		Choices           []ChatChoice `json:"choices,omitempty"`
		Created           int          `json:"created,omitempty"`
		ID                string       `json:"id,omitempty"`
		Model             string       `json:"model,omitempty"`
		Object            string       `json:"object,omitempty"`
		SystemFingerprint string       `json:"system_fingerprint"`
		Usage             *Usage       `json:"usage,omitempty"`
		Error             *Error       `json:"error,omitempty"`
	}

	ChatChoice struct {
		FinishReason string  `json:"finish_reason"`
		Index        int     `json:"index"`
		Logprobs     string  `json:"logprobs"`
		Message      Message `json:"message"`
	}

	Usage struct {
		CompletionTokens       int `json:"completion_tokens"`
		CompletionTokensDetail struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
		PromptTokens       int `json:"prompt_tokens"`
		PromptTokensDetail struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		TotalTokens int `json:"total_tokens"`
	}

	Error struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
		Param   string `json:"param,omitempty"`
		Type    string `json:"type,omitempty"`
	}
	/*
	   {
	     "object": "file",
	     "id": "file-xDqy7hp1VUKBItRWOeIxto3r",
	     "purpose": "assistants",
	     "filename": "goku-schemasys-lock.json",
	     "bytes": 80476,
	     "created_at": 1731908604,
	     "status": "processed",
	     "status_details": null
	   }
	*/
	File struct {
		Object       string  `json:"object"`
		ID           string  `json:"id"`
		Purpose      string  `json:"purpose"`
		Filename     string  `json:"filename"`
		Bytes        int     `json:"bytes"`
		CreatedAt    int     `json:"created_at"`
		Status       string  `json:"status"`
		StatusDetail *string `json:"status_details"`
	}
)

// NewClient creates a new OpenAI client based on HTTP.
func NewClient(ctx context.Context) (Client, error) {

	// Get the API KEY
	apiKey := envutil.GetEnvVarStr("OPENAI_API_KEY")
	if apiKey == "" {
		return Client{}, fmt.Errorf("API Key [OPENAI_API_KEY] not found")
	}

	return NewClientWithKey(ctx, apiKey)

}

func NewClientWithDefaultKey(ctx context.Context) (Client, error) {
	openAPIKey := envutil.GetEnvVarStr("OPENAI_API_KEY")
	if openAPIKey == "" {
		return Client{}, fmt.Errorf("env variable [OPENAI_API_KEY] not found")
	}

	return NewClientWithKey(ctx, openAPIKey)
}

// NewClient creates a new OpenAI client based on HTTP.
func NewClientWithKey(ctx context.Context, apiKey string) (Client, error) {

	if apiKey == "" {
		return Client{}, fmt.Errorf("API Key is empty")
	}

	client := Client{
		httpClient: &http.Client{},
		apiKey:     apiKey,
	}

	return client, nil
}

// func (c Client) ChatStructuredJSON(ctx context.Context, req ChatRequest, respV interface{}) error {

// 	respStr, err := c.Chat(ctx, req)
// 	if err != nil {
// 		return fmt.Errorf("Chatting: %w", err)
// 	}

// 	err = json.Unmarshal([]byte(respStr), respV)
// 	if err != nil {
// 		return fmt.Errorf("Unmarshalling response: %w", err)
// 	}

// 	return nil

// }

// ChatDefault sends a chat request to the OpenAI API.
func (c Client) Chat(ctx context.Context, req ChatRequest) (string, error) {

	var resp ChatDefaultResponse
	err := c.MakeChatRequest(ctx, req, &resp)
	if err != nil {
		return "", fmt.Errorf("Making request: %w", err)
	}

	if resp.Choices == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("No choices in response")
	}

	if resp.Choices[0].Message.Refusal != "" {
		return "", fmt.Errorf("Assistant refused to respond: %s", resp.Choices[0].Message.Refusal)
	}

	if resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("No content in response")
	}

	return resp.Choices[0].Message.Content, nil
}

// Chat sends a chat request to the OpenAI API.
func (c Client) MakeChatRequest(ctx context.Context, reqV ChatRequest, respV interface{}) error {

	req, err := CreateJSONRequest(ctx, c, http.MethodPost, "/v1/chat/completions", reqV)
	if err != nil {
		return fmt.Errorf("Creating new JSON request: %w", err)
	}

	err = DoHTTPRequest(ctx, c, req, respV)
	if err != nil {
		return fmt.Errorf("Making request: %w", err)
	}

	return nil
}

/*
	{
	  "model": "gpt-4o",
	  "name": "Ongoku - Schema Generator Assistant",
	  "description": "A coding assistant that helps users of Ongoku generate and edit schema files.",
	  "tools": [
	    {
	      "type": "code_interpreter"
	    },
	    {
	      "type": "file_search"
	    }
	  ],
	  "tool_resources": {
	    "code_interpreter": {
	      "file_ids": [
	        "file-EuM2VOQG6xzppVAfqz3k6Keu",
	        "file-xDqy7hp1VUKBItRWOeIxto3r"
	      ]
	    },
	    "file_search": {
	      "vector_stores": [
	        {
	          "file_ids": [
	            "file-EuM2VOQG6xzppVAfqz3k6Keu"
	          ]
	        }
	      ]
	    }
	  },
	  "instructions": "You are a coding assistant which is tasked with generating Ongoku-schema files according to the provided spec. The spec (JSON schema for Ongoku-schema file) is provided to you as a file attachment.\n \nThe existing Ongoku-schema of the entire API (which includes built-ins) is provided to you as `goku-schemasys-lock.json`. The schema you generate can use the resources already mentioned in the existing schema.\n\nSometimes, you will be provided with a schema file that you have generated, and the prompt will require you to make edits and re-create the schema file.\n\nContext: Ongoku is low-code backend development framework that relies on code generation to write the boilerplate code for developers. Users provide important information about their API in a schema file, and Ongoku parses that schema file to generate the backend code. \n\nKeep in mind:\n- Each app has services (one or more), each service has entities (zero or more). Entities have fields (one or more), associations (optional), hooks (optional), and actions (optional).\n- CRUD methods for entities DO NOT need to be added , since they are added automatically at later stage. These methods include CRUD methods (Create/Add, Read/Get, Update, Delete), List entity methods, and QueryByText methods.\n- If an entity is associated with another entity, DO NOT explicitly add fields for those entity IDs. Adding an 'association' should be enough. The fields are automatically added at a later stage (based on the associations).\n- In the schema, properties which are of type boolean do not need to be explicitly defined if the intended value is false.\n- Try to make services broad, so many entities can fit into them.\n\nWhen the user provides you with a prompt, share your plan first before generating the schema."
	}
*/

type NewAssistantRequest struct {
	AssistantBase `json:",inline"`
}

type Assistant struct {
	ID            string `json:"id"`
	CreatedAt     int    `json:"created_at"`
	AssistantBase `json:",inline"`
}
type AssistantBase struct {
	Model          Model         `json:"model"` // gpt-4o
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Tools          []Tool        `json:"tools"`
	ToolResources  ToolResources `json:"tool_resources"`
	Instructions   string        `json:"instructions"`
	ResponseFormat interface{}   `json:"response_format,omitempty"`
}

type Tool struct {
	Type string `json:"type"` // code_interpreter, file_search
}

type ToolResources struct {
	CodeInterpreter ToolResourcesCodeInterpreter `json:"code_interpreter"`
	FileSearch      ToolResourcesFileSearch      `json:"file_search,omitempty"`
}
type ToolResourcesCodeInterpreter struct {
	FileIDs []string `json:"file_ids,omitempty"`
}
type ToolResourcesFileSearch struct {
	VectorStores   []ToolResourcesFileSearchVectorStore `json:"vector_stores,omitempty"`
	VectorStoreIDs []string                             `json:"vector_store_ids,omitempty"`
}
type ToolResourcesFileSearchVectorStore struct {
	FileIDs []string `json:"file_ids,omitempty"`
}

func (c Client) CreateAssistant(ctx context.Context, bodyV NewAssistantRequest) (Assistant, error) {

	// Create a JSON request
	req, err := CreateJSONRequest(ctx, c, http.MethodPost, "/v1/assistants", bodyV)
	if err != nil {
		return Assistant{}, fmt.Errorf("Creating new JSON request: %w", err)
	}
	req.Header.Set("OpenAI-Beta", "assistants=v2")

	var resp Assistant
	err = DoHTTPRequest(ctx, c, req, &resp)
	if err != nil {
		return Assistant{}, fmt.Errorf("Making request: %w", err)
	}

	return resp, nil
}

type Thread struct {
	ID         string `json:"id,omitempty"`
	ThreadBase `json:",inline"`
}

type ThreadInput struct {
	Messages   []ThreadMessageInput `json:"messages,omitempty"` // only for initial messages, in the Thread request
	ThreadBase `json:",inline"`
}

type ThreadBase struct{}

type NewThreadAndRunRequest struct {
	AssistantID    string                 `json:"assistant_id"`
	Thread         ThreadInput            `json:"thread,omitempty"`         // Optional (if we want to provide initial messages)
	ToolResources  ToolResources          `json:"tool_resources,omitempty"` // Maybe allows us to provide more files to the assistant?
	ResponseFormat *MessageResponseFormat `json:"response_format,omitempty"`
}

/*
	{
	  "id": "run_abc123",
	  "object": "thread.run",
	  "created_at": 1698107661,
	  "assistant_id": "asst_abc123",
	  "thread_id": "thread_abc123",
	  "status": "completed",
	  "started_at": 1699073476,
	  "expires_at": null,
	  "cancelled_at": null,
	  "failed_at": null,
	  "completed_at": 1699073498,
	  "last_error": null,
	  "model": "gpt-4o",
	  "instructions": null,
	  "tools": [{"type": "file_search"}, {"type": "code_interpreter"}],
	  "metadata": {},
	  "incomplete_details": null,
	  "usage": {
	    "prompt_tokens": 123,
	    "completion_tokens": 456,
	    "total_tokens": 579
	  },
	  "temperature": 1.0,
	  "top_p": 1.0,
	  "max_prompt_tokens": 1000,
	  "max_completion_tokens": 1000,
	  "truncation_strategy": {
	    "type": "auto",
	    "last_messages": null
	  },
	  "response_format": "auto",
	  "tool_choice": "auto",
	  "parallel_tool_calls": true
	}
*/
type ThreadRun struct {
	ID                  string   `json:"id"`
	Object              string   `json:"object"` // Always "thread.run"
	CreatedAt           int      `json:"created_at"`
	AssistantID         string   `json:"assistant_id"`
	ThreadID            string   `json:"thread_id"`
	Status              string   `json:"status"`
	StartedAt           int      `json:"started_at"`
	ExpiresAt           int      `json:"expires_at"`
	CancelledAt         int      `json:"cancelled_at"`
	FailedAt            int      `json:"failed_at"`
	CompletedAt         int      `json:"completed_at"`
	LastError           *Error   `json:"last_error,omitempty"`
	Model               Model    `json:"model"`
	Instructions        string   `json:"instructions"`
	Tools               []Tool   `json:"tools"`
	Metadata            struct{} `json:"metadata"`
	IncompleteDetails   string   `json:"incomplete_details"`
	Usage               Usage    `json:"usage"`
	Temperature         float64  `json:"temperature"`
	TopP                float64  `json:"top_p"`
	MaxPromptTokens     int      `json:"max_prompt_tokens"`
	MaxCompletionTokens int      `json:"max_completion_tokens"`
	TruncationStrategy  struct {
		Type         string `json:"type"`
		LastMessages string `json:"last_messages"`
	} `json:"truncation_strategy"`
	ResponseFormat    interface{} `json:"response_format"`
	ToolChoice        string      `json:"tool_choice"`
	ParallelToolCalls bool        `json:"parallel_tool_calls"`
}

func (c Client) NewChatWithAssistant(ctx context.Context, req NewThreadAndRunRequest) (string, error) {

	run, err := c.CreateThreadAndRunCompleted(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Creating run: %w", err)
	}

	respStr, err := c.GetThreadRunResponse(ctx, run.ThreadID, run.ID)
	if err != nil {
		return "", fmt.Errorf("Getting last message: %w", err)
	}

	return respStr, nil

}

func (c Client) GetThreadRunResponse(ctx context.Context, threadID string, runID string) (string, error) {

	// Get the thread
	msgs, err := c.GetThreadMessages(ctx, threadID)
	if err != nil {
		return "", fmt.Errorf("Getting thread: %w", err)
	}

	if len(msgs) == 0 {
		return "", fmt.Errorf("No messages in thread")
	}

	// The messages may not be ordered.
	// We need to find the message with the given runID
	var msg *ThreadMessage
	for _, m := range msgs {
		if m.RunID == runID && m.Role == Role_Assistant {
			if msg != nil {
				return "", fmt.Errorf("Multiple messages with the same runID [%s] + role [%s]", runID, Role_Assistant)
			}
			msg = &m
			continue
		}
	}
	if msg == nil {
		return "", fmt.Errorf("No message with the given runID")
	}

	if len(msg.Content) == 0 {
		return "", fmt.Errorf("No content in message")
	}

	// We only expect one content
	if len(msg.Content) > 1 {
		return "", fmt.Errorf("Multiple content in message")
	}

	if msg.Content[0].Type != "text" {
		return "", fmt.Errorf("Unexpected content type: %s", msg.Content[0].Type)
	}

	if msg.Content[0].Text.Value == "" {
		return "", fmt.Errorf("Empty value in content")
	}

	return msg.Content[0].Text.Value, nil
}

// func (c Client) NewChatWithAssistantExistingThread(ctx context.Context, req RunRequest) (string, error) {

func (c Client) CreateThreadAndRunCompleted(ctx context.Context, req NewThreadAndRunRequest) (ThreadRun, error) {

	run, err := c.CreateThreadAndRun(ctx, req)
	if err != nil {
		return run, fmt.Errorf("Creating run: %w", err)
	}

	// Wait for the run to complete
	run, err = c.WaitForThreadRunCompletion(ctx, run.ThreadID, run.ID)
	if err != nil {
		return run, fmt.Errorf("Waiting for run completion: %w", err)
	}

	return run, nil
}

func (c Client) WaitForThreadRunCompletion(ctx context.Context, threadID, runID string) (ThreadRun, error) {

	var maxTries = 10
	var sleepTime = 10 * time.Second
	var try = 0
	for {
		try++
		if try > maxTries {
			return ThreadRun{}, fmt.Errorf("Run did not complete after %d tries", maxTries)
		}

		log.Debug(ctx, "Getting thread run", "threadID", threadID, "runID", runID)
		run, err := c.GetThreadRun(ctx, threadID, runID)
		if err != nil {
			return ThreadRun{}, fmt.Errorf("Getting run: %w", err)
		}

		if run.Status == "completed" {
			log.Debug(ctx, "Run completed successfully", "run", run)
			return run, nil
		}

		if run.Status == "failed" {
			return run, fmt.Errorf("Run failed: %v", run.LastError)
		}

		// Sleep for a bit
		log.Debug(ctx, "Run not completed yet. Waiting...", "sleepTime", sleepTime, "status", run.Status)
		time.Sleep(sleepTime)
	}

}

func (c Client) CreateThreadAndRun(ctx context.Context, req NewThreadAndRunRequest) (ThreadRun, error) {

	// Create a JSON request
	httpReq, err := CreateJSONRequest(ctx, c, http.MethodPost, "v1/threads/runs", req)
	if err != nil {
		return ThreadRun{}, fmt.Errorf("Creating new JSON request: %w", err)
	}

	// Set the OpenAI-Beta header
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	// Make the request
	var resp ThreadRun
	err = DoHTTPRequest(ctx, c, httpReq, &resp)
	if err != nil {
		return ThreadRun{}, fmt.Errorf("Making request: %w", err)
	}

	return resp, nil
}

type ThreadRunInput struct {
	AssistantID        string                 `json:"assistant_id"`
	Model              Model                  `json:"model"` // If provided, will override the assistant's model
	AdditionalMessages []ThreadMessageInput   `json:"additional_messages,omitempty"`
	ResponseFormat     *MessageResponseFormat `json:"response_format,omitempty"`
}

func (c Client) NewThreadRun(ctx context.Context, threadID string, req ThreadRunInput) (ThreadRun, error) {

	// Create a JSON request
	httpReq, err := CreateJSONRequest(ctx, c, http.MethodPost, fmt.Sprintf("v1/threads/%s/runs", threadID), req)
	if err != nil {
		return ThreadRun{}, fmt.Errorf("Creating new JSON request: %w", err)
	}

	// Set the OpenAI-Beta header
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	// Make the request
	var resp ThreadRun
	err = DoHTTPRequest(ctx, c, httpReq, &resp)
	if err != nil {
		return ThreadRun{}, fmt.Errorf("Making request: %w", err)
	}

	return resp, nil
}

func (c Client) NewThreadRunComplete(ctx context.Context, threadID string, req ThreadRunInput) (ThreadRun, error) {

	log.Debug(ctx, "Creating new thread run", "threadID", threadID)
	run, err := c.NewThreadRun(ctx, threadID, req)
	if err != nil {
		return run, fmt.Errorf("Creating new thread run: %w", err)
	}

	// Wait for the run to complete
	log.Debug(ctx, "Waiting for thread run completion", "threadID", threadID, "runID", run.ID)
	run, err = c.WaitForThreadRunCompletion(ctx, threadID, run.ID)
	if err != nil {
		return run, fmt.Errorf("Waiting for run completion: %w", err)
	}

	return run, nil
}

func (c Client) GetThreadRun(ctx context.Context, threadID, runID string) (ThreadRun, error) {

	// Create a JSON request
	httpReq, err := CreateAuthorizedHTTPRequest(ctx, c, http.MethodGet, fmt.Sprintf("v1/threads/%s/runs/%s", threadID, runID), nil)
	if err != nil {
		return ThreadRun{}, fmt.Errorf("Creating new authorized HTTP request: %w", err)
	}

	// Set the OpenAI-Beta header
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	// Make the request
	var resp ThreadRun
	err = DoHTTPRequest(ctx, c, httpReq, &resp)
	if err != nil {
		return ThreadRun{}, fmt.Errorf("Making request: %w", err)
	}

	return resp, nil
}

func (c Client) NewThread(ctx context.Context, req ThreadInput) (Thread, error) {
	// Create a JSON request
	httpReq, err := CreateJSONRequest(ctx, c, http.MethodPost, "v1/threads", req)
	if err != nil {
		return Thread{}, fmt.Errorf("Creating new JSON request: %w", err)
	}
	// Set the OpenAI-Beta header
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	// Make the request
	var resp Thread
	err = DoHTTPRequest(ctx, c, httpReq, &resp)
	if err != nil {
		return Thread{}, fmt.Errorf("Making request: %w", err)
	}
	return resp, nil
}

func (c Client) GetThread(ctx context.Context, threadID string) (Thread, error) {

	// Create a JSON request
	httpReq, err := CreateAuthorizedHTTPRequest(ctx, c, http.MethodGet, fmt.Sprintf("v1/threads/%s", threadID), nil)
	if err != nil {
		return Thread{}, fmt.Errorf("Creating new authorized HTTP request: %w", err)
	}
	// Set the OpenAI-Beta header
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	// Make the request
	var resp Thread
	err = DoHTTPRequest(ctx, c, httpReq, &resp)
	if err != nil {
		return Thread{}, fmt.Errorf("Making request: %w", err)
	}

	return resp, nil
}

type ThreadMessageInput struct {
	ThreadMessageBase `json:",inline"`
	Content           string `json:"content"`
}

type ThreadMessage struct {
	ThreadMessageBase `json:",inline"`
	Content           []ThreadMessageContent `json:"content"`
	ID                string                 `json:"id"`
	CreatedAt         int                    `json:"created_at"`
	AssistantID       string                 `json:"assistant_id"`
	ThreadID          string                 `json:"thread_id"`
	RunID             string                 `json:"run_id"`
}

type ThreadMessageBase struct {
	Role Role `json:"role"`
}

type (
	ThreadMessageContent struct {
		Type string                   `json:"type"`
		Text ThreadMessageContentText `json:"text"`
	}

	ThreadMessageContentText struct {
		Value       string        `json:"value"`
		Annotations []interface{} `json:"annotations"`
	}
)

type ThreadMessagesResponse struct {
	Object string          `json:"object"`
	Data   []ThreadMessage `json:"data"`
}

func (c Client) GetThreadMessages(ctx context.Context, threadID string) ([]ThreadMessage, error) {

	// Create a JSON request
	httpReq, err := CreateAuthorizedHTTPRequest(ctx, c, http.MethodGet, fmt.Sprintf("v1/threads/%s/messages", threadID), nil)
	if err != nil {
		return nil, fmt.Errorf("Creating new authorized HTTP request: %w", err)
	}
	// Set the OpenAI-Beta header
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	// Make the request
	var resp ThreadMessagesResponse
	err = DoHTTPRequest(ctx, c, httpReq, &resp)
	if err != nil {
		return nil, fmt.Errorf("Making request: %w", err)
	}

	return resp.Data, nil

}

func (c Client) UploadFile(ctx context.Context, fileName string, fileData []byte) (File, error) {

	var ret File

	req, err := CreateUploadFileHTTPRequest(ctx, c, fileName, fileData)
	if err != nil {
		return ret, fmt.Errorf("Getting upload file request: %w", err)
	}

	// Make the request
	err = DoHTTPRequest(ctx, c, req, &ret)
	if err != nil {
		return ret, fmt.Errorf("Making request: %w", err)
	}

	return ret, nil
}

/*
	{
	  "id": "asst_e5qL6DqZr0gD2PpYjpRAg4BQ",
	  "object": "assistant.deleted",
	  "deleted": true
	}
*/
type DeleteResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

func (c Client) DeleteAssistant(ctx context.Context, id string) error {
	log.Debug(ctx, "Deleting OpenAI Assistant", "id", id)

	if id == "" {
		log.Warn(ctx, "OpenAI delete called on empty assistant ID")
		return nil
	}

	err := c.Delete(ctx, fmt.Sprintf("v1/assistants/%s", id))
	if err != nil {
		return err
	}
	return nil
}

func (c Client) DeleteFile(ctx context.Context, id string) error {
	log.Debug(ctx, "Deleting OpenAI File", "id", id)

	if id == "" {
		log.Warn(ctx, "OpenAI delete called on empty file ID")
		return nil
	}

	err := c.Delete(ctx, fmt.Sprintf("v1/files/%s", id))
	if err != nil {
		return err
	}
	return nil
}

func (c Client) Delete(ctx context.Context, path string) error {

	// Create a HTTP request
	req, err := CreateDeleteHTTPRequest(ctx, c, path)
	if err != nil {
		return fmt.Errorf("Creating new delete HTTP request: %w", err)
	}

	// Add the OpenAI-Beta header
	req.Header.Set("OpenAI-Beta", "assistants=v2")

	// Make the request
	var resp DeleteResponse
	err = DoHTTPRequest(ctx, c, req, &resp)
	if err != nil {
		return fmt.Errorf("Making request: %w", err)
	}

	return nil
}

func CreateDeleteHTTPRequest(ctx context.Context, c Client, path string) (*http.Request, error) {

	// Create the request
	req, err := CreateAuthorizedHTTPRequest(ctx, c, http.MethodDelete, path, nil)
	if err != nil {
		return nil, fmt.Errorf("Creating new authorized HTTP request: %w", err)
	}

	return req, nil
}

func CreateUploadFileHTTPRequest(ctx context.Context, c Client, fileName string, fileData []byte) (*http.Request, error) {

	path := "v1/files"
	method := http.MethodPost

	// Create a buffer to store our multipart form data
	var reqBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&reqBody)

	// Add the "purpose" field
	err := multipartWriter.WriteField("purpose", "assistants")
	if err != nil {
		return nil, fmt.Errorf("could not write purpose field: %w", err)
	}

	// Create a form file field and copy the file content into it
	fileWriter, err := multipartWriter.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("could not create form file: %w", err)
	}
	_, err = fileWriter.Write(fileData)
	if err != nil {
		return nil, fmt.Errorf("could not write file content: %w", err)
	}

	// Close the multipart writer to finalize the request body
	err = multipartWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("could not close multipart writer: %w", err)
	}

	req, err := CreateAuthorizedHTTPRequest(ctx, c, method, path, &reqBody)
	if err != nil {
		return nil, fmt.Errorf("Creating new authorized HTTP request: %w", err)
	}

	// Edit the request
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	return req, nil
}

func CreateJSONRequest(ctx context.Context, c Client, method, path string, bodyV interface{}) (*http.Request, error) {

	// Get the body
	bodyRdr := new(bytes.Buffer)
	err := json.NewEncoder(bodyRdr).Encode(bodyV)
	if err != nil {
		return nil, fmt.Errorf("Encoding body: %w", err)
	}

	req, err := CreateAuthorizedHTTPRequest(ctx, c, method, path, bodyRdr)
	if err != nil {
		return nil, fmt.Errorf("Creating authorized HTTP request: %w", err)
	}

	// Add header
	req.Header.Set("Content-Type", "application/json")

	log.Debug(ctx, "Created JSON Request", "body", bodyRdr.String())

	return req, nil
}

func CreateAuthorizedHTTPRequest(ctx context.Context, c Client, method, path string, bodyRdr io.Reader) (*http.Request, error) {

	// Create a HTTP request
	baseURL := "https://api.openai.com"
	fullURL, err := url.JoinPath(baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("Joining URL path: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyRdr)
	if err != nil {
		return nil, fmt.Errorf("Creating new request: %w", err)
	}

	// Set the auth headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	log.Debug(ctx, "Created Request", "method", method, "url", fullURL, "headers", jsonutil.MustPrettyPrint(req.Header))

	return req, nil
}

func DoHTTPRequest(ctx context.Context, c Client, req *http.Request, resp interface{}) error {

	// Make the request
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Making request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read the response
	respBodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("Reading response: %w", err)
	}

	log.Debug(ctx, "Response Body", "data", string(respBodyBytes))

	// Decode the response
	err = json.Unmarshal(respBodyBytes, resp)
	if err != nil {
		return fmt.Errorf("Decoding response: %w", err)
	}

	log.None(ctx, "Parsed Response", "data", jsonutil.MustPrettyPrint(resp))

	// Check the status code
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		log.Warn(ctx, "Bad status code", "status", httpResp.Status)
		return fmt.Errorf("Bad status code: %s", httpResp.Status)
	}

	return nil

}

// // MakeRequest makes a request to the OpenAI API.
// func MakeRequest[ReqT any, RespT any](ctx context.Context, c Client, method string, path string, data ReqT, respV RespT) error {

// 	// Create a HTTP request
// 	baseURL := "https://api.openai.com"
// 	fullURL, err := url.JoinPath(baseURL, path)
// 	if err != nil {
// 		return fmt.Errorf("Joining URL path: %w", err)
// 	}

// 	// Get io.Reader for the data
// 	dataBytes, err := json.Marshal(data)
// 	if err != nil {
// 		return fmt.Errorf("Marshalling data: %w", err)
// 	}

// 	// Create the request
// 	buff := bytes.NewBuffer(dataBytes)

// 	req, err := http.NewRequestWithContext(ctx, method, fullURL, buff)
// 	if err != nil {
// 		return fmt.Errorf("Creating new request: %w", err)
// 	}

// 	// Set the headers
// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
// 	req.Header.Set("Content-Type", "application/json")

// 	log.Debug(ctx, "Making request", "method", method, "url", fullURL, "headers", jsonutil.MustPrettyPrint(req.Header))
// 	log.Debug(ctx, "Request", "request", jsonutil.MustPrettyPrint(data))

// 	// Make the request
// 	resp, err := c.httpClient.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("Making request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	// Decode the response
// 	decoder := json.NewDecoder(resp.Body)
// 	err = decoder.Decode(respV)
// 	if err != nil {
// 		return fmt.Errorf("Decoding response: %w", err)
// 	}

// 	log.Debug(ctx, "Response", "response", jsonutil.MustPrettyPrint(respV))

// 	// Check the status code
// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
// 		log.Warn(ctx, "Bad status code", "status", resp.Status)
// 		return fmt.Errorf("Bad status code: %s", resp.Status)
// 	}

// 	return nil

// }
