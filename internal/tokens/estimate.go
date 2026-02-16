package tokens

// EstimateTokens provides a simple heuristic for estimating token count
// from text content. This uses ~4 characters per token for English text,
// which is a reasonable approximation for most LLM tokenizers without
// adding heavy dependencies like tiktoken.
//
// For more accurate estimates, integrate a proper tokenizer library,
// but this heuristic is sufficient for budgeting context windows.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	// simple heuristic: ~4 characters per token for English
	// this works reasonably well for GPT-style tokenizers
	return (len(text) + 3) / 4
}

// EstimateTokensWithChildren estimates tokens for content plus child descriptions
func EstimateTokensWithChildren(content string, childDescriptions map[string]string) int {
	total := EstimateTokens(content)

	for _, desc := range childDescriptions {
		total += EstimateTokens(desc)
	}

	return total
}
