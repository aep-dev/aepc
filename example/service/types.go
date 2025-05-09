package service

import (
	"encoding/json"
	"fmt"

	bpb "github.com/aep-dev/aepc/example/bookstore/v1"
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
