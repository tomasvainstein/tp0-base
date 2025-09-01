package common

import "fmt"

type Bet struct {
	Nombre     string `json:"nombre"`
	Apellido   string `json:"apellido"`
	Documento  string `json:"documento"`
	Nacimiento string `json:"nacimiento"`
	Numero     string `json:"numero"`
}

func NewBet(nombre, apellido, documento, nacimiento, numero string) *Bet {
	return &Bet{
		Nombre:     nombre,
		Apellido:   apellido,
		Documento:  documento,
		Nacimiento: nacimiento,
		Numero:     numero,
	}
}

func (b *Bet) Validate() error {
	if b.Nombre == "" {
		return fmt.Errorf("nombre es requerido")
	}
	if b.Apellido == "" {
		return fmt.Errorf("apellido es requerido")
	}
	if b.Documento == "" {
		return fmt.Errorf("documento es requerido")
	}
	if b.Nacimiento == "" {
		return fmt.Errorf("nacimiento es requerido")
	}
	if b.Numero == "" {
		return fmt.Errorf("numero es requerido")
	}
	return nil
}
