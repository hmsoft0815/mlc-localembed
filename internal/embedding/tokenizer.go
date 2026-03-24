// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package embedding

import (
	"fmt"

	"github.com/daulet/tokenizers"
)

// TokenizerWrapper wraps the daulet/tokenizers implementation.
type TokenizerWrapper struct {
	tk *tokenizers.Tokenizer
}

// NewTokenizer loads a tokenizer from a tokenizer.json file.
func NewTokenizer(path string) (*TokenizerWrapper, error) {
	tk, err := tokenizers.FromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer from %s: %w", path, err)
	}
	return &TokenizerWrapper{tk: tk}, nil
}

// Encode encodes a single string into token IDs, attention mask, and type IDs.
// It handles truncation and padding to maxLen.
func (t *TokenizerWrapper) Encode(text string, maxLen int) (ids, mask, typeIds []int64, err error) {
	// Use EncodeWithOptionsErr to get all required attributes
	encoding, err := t.tk.EncodeWithOptionsErr(
		text,
		true,
		tokenizers.WithReturnAttentionMask(),
		tokenizers.WithReturnTypeIDs(),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("tokenization failed: %w", err)
	}

	allIds := encoding.IDs
	allMask := encoding.AttentionMask
	allTypeIds := encoding.TypeIDs

	length := len(allIds)
	if length > maxLen {
		length = maxLen
	}

	ids = make([]int64, maxLen)
	mask = make([]int64, maxLen)
	typeIds = make([]int64, maxLen)

	for i := 0; i < length; i++ {
		ids[i] = int64(allIds[i])
		if i < len(allMask) {
			mask[i] = int64(allMask[i])
		}
		if i < len(allTypeIds) {
			typeIds[i] = int64(allTypeIds[i])
		}
	}

	return ids, mask, typeIds, nil
}

// Close releases the underlying tokenizer resources.
func (t *TokenizerWrapper) Close() {
	if t.tk != nil {
		t.tk.Close()
	}
}
