// Package architecture verifica la dirección de imports entre capas y módulos
// (PLANTILLA_BACKEND_GO.md, sección 4). Compilar sin ciclos no prueba esto.
package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "service-restaurant"

// Infraestructura que nunca debe filtrarse a dominio, aplicación ni shared.
var infraLibs = []string{"database/sql", "net/http", "github.com/jackc/", "github.com/pressly/goose"}

var (
	moduleRe = regexp.MustCompile(`^internal/modules/([^/]+)/(domain|application|infrastructure|web)(?:/([^/]+))?`)
)

// violation devuelve el motivo si el paquete pkg (ruta relativa a la raíz) no puede importar imp.
func violation(pkg, imp string) string {
	internal := strings.HasPrefix(imp, modulePath+"/")
	rel := strings.TrimPrefix(imp, modulePath+"/")
	std := !internal && !strings.Contains(strings.SplitN(imp, "/", 2)[0], ".")

	hasInfra := func() bool {
		for _, l := range infraLibs {
			if imp == l || strings.HasPrefix(imp, l) {
				return true
			}
		}
		return false
	}
	underAny := func(prefixes ...string) bool {
		for _, p := range prefixes {
			if rel == p || strings.HasPrefix(rel, p+"/") {
				return true
			}
		}
		return false
	}

	// --- módulos de negocio
	if m := moduleRe.FindStringSubmatch(pkg); m != nil {
		mod, layer, sub := m[1], m[2], m[3]
		own := "internal/modules/" + mod
		if internal && strings.HasPrefix(rel, "internal/modules/") && !underAny(own) {
			return "importa internos de otro módulo"
		}
		if internal && underAny("internal/bootstrap") {
			return "importa bootstrap"
		}
		switch layer {
		case "domain":
			if hasInfra() {
				return "el dominio no importa SQL, HTTP ni drivers"
			}
			if internal && underAny(own+"/application", own+"/infrastructure", own+"/web", "internal/platform", "internal/shared/application") {
				return "el dominio no importa aplicación, infraestructura, web, plataforma ni shared/application"
			}
			switch sub {
			case "vo", "domainerror":
				if !std {
					return sub + " solo puede importar la biblioteca estándar"
				}
			case "model":
				if internal && !underAny(own+"/domain/vo", own+"/domain/domainerror", "internal/shared/kernel") {
					return "model solo importa vo y domainerror (no policy ni service)"
				}
			case "policy", "service":
				if internal && !underAny(own+"/domain/model", own+"/domain/vo", own+"/domain/domainerror", "internal/shared/kernel") {
					return sub + " solo importa model, vo y domainerror"
				}
			}
		case "application":
			if hasInfra() {
				return "la aplicación no importa SQL, HTTP ni drivers"
			}
			if internal && underAny(own+"/infrastructure", own+"/web", "internal/platform") {
				return "la aplicación no importa infraestructura, web ni plataforma"
			}
		case "infrastructure":
			// puede importar aplicación, dominio, plataforma y shared. La web (adaptador de
			// entrada) solo la conoce infrastructure/config, que compone el módulo.
			if internal && underAny(own+"/web") && !(sub == "config") {
				return "solo infrastructure/config importa la web del módulo"
			}
		case "web":
			// adaptador de entrada: depende de los puertos/DTO de aplicación, shared y plataforma web
			if internal && underAny(own+"/domain") {
				return "la web usa DTO y puertos de aplicación, no importa el dominio"
			}
			if internal && underAny(own+"/infrastructure") {
				return "la web no importa infraestructura"
			}
		}
		return ""
	}

	// --- shared: núcleo puro
	if strings.HasPrefix(pkg, "internal/shared/") {
		if hasInfra() {
			return "shared no importa SQL, HTTP ni drivers"
		}
		if internal && underAny("internal/modules", "internal/platform", "internal/bootstrap") {
			return "shared no importa módulos, plataforma ni bootstrap"
		}
		return ""
	}

	// --- platform: técnica transversal que no conoce el negocio
	if strings.HasPrefix(pkg, "internal/platform/") {
		if internal && underAny("internal/modules", "internal/bootstrap") {
			return "platform no importa módulos ni bootstrap"
		}
		return ""
	}

	// --- bootstrap: solo ensambla; de un módulo toma únicamente su composición
	if pkg == "internal/bootstrap" {
		if internal && underAny("internal/modules") && !strings.HasSuffix(rel, "/infrastructure/config") {
			return "bootstrap solo importa infrastructure/config de un módulo"
		}
		return ""
	}

	// --- cmd: arranca y delega
	if strings.HasPrefix(pkg, "cmd/") {
		if internal && underAny("internal/modules") {
			return "cmd no importa módulos de negocio"
		}
	}
	return ""
}

func findRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no se encontró go.mod")
		}
		dir = parent
	}
}

// El código real del repositorio cumple las reglas.
func TestImportDirection(t *testing.T) {
	root := findRoot(t)
	fset := token.NewFileSet()
	checked := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "tests", "testdata", "graphify-out", "migrations":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		pkg, _ := filepath.Rel(root, filepath.Dir(path))
		pkg = filepath.ToSlash(pkg)
		for _, is := range f.Imports {
			imp, _ := strconv.Unquote(is.Path.Value)
			if why := violation(pkg, imp); why != "" {
				t.Errorf("%s importa %q: %s", filepath.ToSlash(path[len(root)+1:]), imp, why)
			}
		}
		checked++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 10 {
		t.Fatalf("solo se revisaron %d archivos; ¿cambió la estructura?", checked)
	}
}

// Las reglas no son vacuas: casos sintéticos que DEBEN rechazarse o aceptarse.
func TestRulesDetectViolations(t *testing.T) {
	const m = modulePath + "/internal/modules/catalogo"
	cases := []struct {
		pkg, imp string
		reject   bool
	}{
		{"internal/modules/catalogo/domain/vo", "database/sql", true},
		{"internal/modules/catalogo/domain/vo", m + "/domain/model", true},
		{"internal/modules/catalogo/domain/vo", "errors", false},
		{"internal/modules/catalogo/domain/model", m + "/domain/vo", false},
		{"internal/modules/catalogo/domain/model", m + "/domain/policy", true},
		{"internal/modules/catalogo/domain/policy", m + "/domain/model", false},
		{"internal/modules/catalogo/domain/service", m + "/application/port/outbound", true},
		{"internal/modules/catalogo/application/usecase/command", "database/sql", true},
		{"internal/modules/catalogo/application/usecase/command", "net/http", true},
		{"internal/modules/catalogo/application/usecase/command", m + "/infrastructure/config", true},
		{"internal/modules/catalogo/application/usecase/command", modulePath + "/internal/platform/transaction", true},
		{"internal/modules/catalogo/application/usecase/command", m + "/domain/model", false},
		{"internal/modules/catalogo/application/usecase/command", modulePath + "/internal/shared/kernel/result", false},
		{"internal/modules/catalogo/infrastructure/config", modulePath + "/internal/platform/transaction", false},
		{"internal/modules/catalogo/infrastructure/config", m + "/application/usecase/command", false},
		{"internal/modules/catalogo/web/controller", modulePath + "/internal/modules/pedidos/domain/model", true},
		{"internal/modules/catalogo/web/controller", modulePath + "/internal/bootstrap", true},
		{"internal/modules/catalogo/web/controller", m + "/application/port/inbound", false},
		{"internal/modules/catalogo/web/controller", modulePath + "/internal/platform/web/handler", false},
		{"internal/modules/catalogo/web/mapper", m + "/domain/model", true},
		{"internal/modules/catalogo/web/controller", m + "/infrastructure/persistence/read/adapter", true},
		{"internal/modules/catalogo/infrastructure/config", m + "/web/controller", false},
		{"internal/modules/catalogo/infrastructure/persistence/read/adapter", m + "/web/controller", true},
		{"internal/modules/catalogo/application/usecase/query", m + "/web/dto/response", true},
		{"internal/modules/catalogo/domain/model", m + "/web/dto/response", true},
		{"internal/shared/kernel/result", "net/http", true},
		{"internal/shared/application/pagination", modulePath + "/internal/modules/catalogo/domain/vo", true},
		{"internal/platform/transaction", modulePath + "/internal/modules/catalogo/domain/model", true},
		{"internal/platform/database", "github.com/jackc/pgx/v5/stdlib", false},
		{"internal/bootstrap", m + "/infrastructure/config", false},
		{"internal/bootstrap", m + "/domain/model", true},
		{"cmd/api", m + "/infrastructure/config", true},
		{"cmd/api", modulePath + "/internal/bootstrap", false},
	}
	for _, c := range cases {
		got := violation(c.pkg, c.imp) != ""
		if got != c.reject {
			t.Errorf("%s -> %s: rechazado=%v, esperado=%v", c.pkg, c.imp, got, c.reject)
		}
	}
}

// El módulo atencion no puede depender de los internos de identidad (la coordinación entre
// módulos ocurre en la composición).
func TestAtencionNoImportaOtrosModulos(t *testing.T) {
	paquetes := []string{
		"internal/modules/atencion/domain/vo",
		"internal/modules/atencion/domain/policy",
		"internal/modules/atencion/application/usecase/command",
		"internal/modules/atencion/infrastructure/persistence/write/repository",
		"internal/modules/atencion/web/controller",
	}
	for _, pkg := range paquetes {
		if violation(pkg, modulePath+"/internal/modules/identidad/domain/policy") == "" {
			t.Errorf("%s no debe poder importar identidad", pkg)
		}
	}
}
