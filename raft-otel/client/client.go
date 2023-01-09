package client

type Client struct {
}

func New() *Client {
	return &Client{}
}

func (s *Client) Run() error {
	return nil
}
