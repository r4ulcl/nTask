package api

func incorrectOauth(clientOauthKey, oauthToken string) bool {
	return clientOauthKey != oauthToken
}

func incorrectOauthWorker(clientOauthKey, oauthTokenWorkers string) bool {
	return clientOauthKey != oauthTokenWorkers
}
