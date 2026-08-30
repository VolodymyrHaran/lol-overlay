package messaging

type GameDeadLetterEvent struct {
	EventMetadata

	OriginalSubject string `json:"originalSubject"`
	OriginalPayload []byte `json:"originalPayload"`

	Error         string `json:"error"`
	DeliveryCount uint64 `json:"deliveryCount"`

	SourceStream   string `json:"sourceStream"`
	StreamSequence uint64 `json:"streamSequence"`
	Consumer       string `json:"consumer"`
}
