package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("starting server on port %s", port)

	client := openai.NewClient()

	http.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		// validate HTTP method
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, "method not allowed")
			return
		}

		// validate content type
		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "invalid content type: must be an image")
			return
		}

		// limit file size to 10MB
		file := http.MaxBytesReader(w, r.Body, 10<<20)
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("error reading image:\n%v", err)
			fmt.Fprintf(w, "error reading image, check logs for more details")
			return
		}

		prompt := "Extract text from the image using OCR (Optical Character Recognition). Process the image to accurately detect and extract the text content. The output will consist only of the extracted text in a copyable format, without any additional responses, explanations, or comments. Focus solely on providing the requested content."

		base64ImageData := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(fileBytes))

		chatCompletion, err := client.Chat.Completions.New(r.Context(), openai.ChatCompletionNewParams{
			Model: openai.F(openai.ChatModelGPT4oMini),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessageParts(
					openai.TextPart(prompt),
					openai.ImagePart(base64ImageData),
				),
			}),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("error processing image:\n%v", err)
			fmt.Fprintf(w, "error processing image, check logs for more details")
			return
		}

		log.Printf("successfully processed image")
		fmt.Fprint(w, chatCompletion.Choices[0].Message.Content)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
