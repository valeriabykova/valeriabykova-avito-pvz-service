package oapi

type GetPvzReceptionItem struct {
	Reception Reception `json:"reception"`
	Products  []Product `json:"products"`
}

type GetPvzResponseItem struct {
	Pvz        PVZ                   `json:"pvz"`
	Receptions []GetPvzReceptionItem `json:"receptions"`
}
