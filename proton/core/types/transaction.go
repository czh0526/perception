package types

type Transaction struct {
	data txdata
}

type txdata struct {
	Dummy string `json:"dummy" gencodec:"required"`
}
