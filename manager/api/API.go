package api

// login function for user
func incorrectOauth(clientOauthKey, oauthToken string) bool {
	return clientOauthKey != oauthToken
}

// login function for workers
func incorrectOauthWorker(clientOauthKey, oauthTokenWorkers string) bool {
	return clientOauthKey != oauthTokenWorkers
}
