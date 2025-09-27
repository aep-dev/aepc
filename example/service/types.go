package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	bpb "github.com/aep-dev/aepc/example/bookstore/v1"
	"google.golang.org/protobuf/proto"
)

type SerializableBook struct {
	*bpb.Book
	AuthorSerialized string
	IsbnSerialized   string
}

func NewSerializableBook(b *bpb.Book) (*SerializableBook, error) {
	authorSerialized, err := json.Marshal(b.Author)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize author: %v", err)
	}
	isbnSerialized, err := json.Marshal(b.Isbn)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize author: %v", err)
	}
	return &SerializableBook{
		Book:             b,
		AuthorSerialized: string(authorSerialized),
		IsbnSerialized:   string(isbnSerialized),
	}, nil
}

func UnmarshalIntoBook(authorsSerialized, isbnSerialized string, b *bpb.Book) error {
	err := json.Unmarshal([]byte(authorsSerialized), &b.Author)
	if err != nil {
		return fmt.Errorf("failed to deserialize authors: %v", err)
	}

	err = json.Unmarshal([]byte(isbnSerialized), &b.Isbn)
	if err != nil {
		return fmt.Errorf("failed to deserialize isbn: %v", err)
	}
	return nil
}

// GenerateETag generates an ETag for a protobuf message based on its content
func GenerateETag(msg proto.Message) (string, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message for ETag: %v", err)
	}

	hash := md5.Sum(data)
	return `"` + hex.EncodeToString(hash[:]) + `"`, nil
}

// ValidateETag compares the provided ETag with the current resource ETag
func ValidateETag(providedETag, currentETag string) bool {
	// Remove quotes from both ETags if present for comparison
	cleanProvided := strings.Trim(providedETag, `"`)
	cleanCurrent := strings.Trim(currentETag, `"`)
	return cleanProvided == cleanCurrent
}
