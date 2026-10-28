package vo

import "testing"

func TestTipoPedido(t *testing.T) {
	for _, ok := range []string{"SALON", " LLEVAR ", "DELIVERY"} {
		if _, err := NewTipoPedido(ok); err != nil {
			t.Errorf("%q debía ser válido: %v", ok, err)
		}
	}
	for _, mal := range []string{"", "salon", "MOSTRADOR"} {
		if _, err := NewTipoPedido(mal); err == nil {
			t.Errorf("%q debía rechazarse", mal)
		}
	}
	if tp, _ := NewTipoPedido(" LLEVAR "); tp != TipoLlevar || tp.String() != "LLEVAR" {
		t.Errorf("no normalizó: %q", tp)
	}
}

func TestComensales(t *testing.T) {
	for _, n := range []int{1, 2, 500} {
		if c, err := NewComensales(n); err != nil || c.Int() != n {
			t.Errorf("%d debía ser válido: %v", n, err)
		}
	}
	for _, n := range []int{0, -1, 501} {
		if _, err := NewComensales(n); err == nil {
			t.Errorf("%d debía rechazarse", n)
		}
	}
}

func TestCantidad(t *testing.T) {
	for _, ok := range []string{"1", "2.5", " 0.001 ", "99999999999.999"} {
		if _, err := NewCantidad(ok); err != nil {
			t.Errorf("%q debía ser válida: %v", ok, err)
		}
	}
	for _, mal := range []string{"", "0", "0.000", "-1", "+1", "1.2345", "1,5", "abc", "123456789012", "1e3"} {
		if _, err := NewCantidad(mal); err == nil {
			t.Errorf("%q debía rechazarse", mal)
		}
	}
	if c, _ := NewCantidad(" 2.50 "); c.String() != "2.50" {
		t.Errorf("no recortó: %q", c.String())
	}
}

func TestDinero(t *testing.T) {
	for _, ok := range []string{"0", "0.00", "5", "12.34", "999999999999.99"} {
		if _, err := NewDinero(ok); err != nil {
			t.Errorf("%q debía ser válido: %v", ok, err)
		}
	}
	for _, mal := range []string{"", "-1", "1.234", "x", "1000000000000"} {
		if _, err := NewDinero(mal); err == nil {
			t.Errorf("%q debía rechazarse", mal)
		}
	}
	if d, _ := NewDinero("0.00"); !d.EsCero() {
		t.Error("0.00 es cero")
	}
	if d, _ := NewDinero("0.01"); d.EsCero() {
		t.Error("0.01 no es cero")
	}
}

func TestDescuentoCabe(t *testing.T) {
	casos := []struct {
		nombre, cantidad, precio, descuento string
		quiere                              bool
	}{
		{"igual al importe", "2", "10.000000", "20.00", true},
		{"un centavo más", "2", "10.000000", "20.01", false},
		{"cero siempre cabe", "1", "0.000000", "0", true},
		{"redondea el importe a 2 decimales", "3", "3.333333", "10.00", true},
		{"el redondeo no regala un centavo", "3", "3.333333", "10.01", false},
		{"cantidad fraccionaria", "0.5", "9.990000", "5.00", true},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			cant, err := NewCantidad(c.cantidad)
			if err != nil {
				t.Fatal(err)
			}
			d, err := NewDinero(c.descuento)
			if err != nil {
				t.Fatal(err)
			}
			if got := DescuentoCabe(cant, c.precio, d); got != c.quiere {
				t.Errorf("DescuentoCabe(%s × %s, %s) = %v, quería %v", c.cantidad, c.precio, c.descuento, got, c.quiere)
			}
		})
	}
	cant, _ := NewCantidad("1")
	d, _ := NewDinero("0")
	if DescuentoCabe(cant, "no-es-un-numero", d) {
		t.Error("un precio ilegible no debe aceptar el descuento")
	}
}
