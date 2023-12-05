package api

// incorrectOauth compares the client's OAuth key with the OAuth token and returns true if they do not match.
func incorrectOauth(clientOauthKey, oauthToken string) bool {
	return clientOauthKey != oauthToken
}

// incorrectOauthWorker compares the client's OAuth key with the OAuth token for workers and returns true if they do not match.
func incorrectOauthWorker(clientOauthKey, oauthTokenWorkers string) bool {
	return clientOauthKey != oauthTokenWorkers
}
