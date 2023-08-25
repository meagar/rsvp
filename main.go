package main

import (
	"bufio"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed templates
var templateFS embed.FS
var rootTemplate *template.Template
var db sql.DB

func init() {
	loadEnv()
	loadTemplates()
	connectDB()
}

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			log.Fatalf("Malformed line in .env: %s", line)
		}

		if _, ok := os.LookupEnv(parts[0]); !ok {
			log.Printf("ENV[%s] is unset: Using .env value \"%s\"", parts[0], parts[1])
			os.Setenv(parts[0], parts[1])
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}

func loadTemplates() {
	rootTemplate = template.New("")
	fs.WalkDir(templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if !d.IsDir() {
			name := strings.TrimPrefix(path, "templates/")
			name = strings.TrimSuffix(name, ".tmpl")
			log.Println("Template", name)
			f, err := templateFS.Open(path)
			if err != nil {
				log.Fatal(err)
			}
			bytes, err := ioutil.ReadAll(f)
			if err != nil {
				log.Fatal(err)
			}

			rootTemplate.New(name).Parse(string(bytes))
		}
		return nil
	})

	log.Println(rootTemplate.DefinedTemplates())
}

func connectDB() *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	return conn
}

func main() {
	port := fetchEnv("PORT")
	log.Println("Running on port", port)

	db := connectDB()
	defer db.Close(context.Background())

	// pg_addr := fetchEnv("PG_ADDR")
	// log.Println("Connecting to database", pg_addr)

	adminPath := fetchEnvDef("ADMIN_PATH", "/admin/")
	log.Printf("Serving admin site from %s", adminPath)

	http.Handle(adminPath, &AdminHandler{})
	http.Handle("/", &Handler{db: db})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func fetchEnv(name string) string {
	if value, ok := os.LookupEnv(name); !ok {
		fmt.Printf("ENV variable %s is not set\n", name)
		os.Exit(1)
		return ""
	} else {
		return value
	}
}

func fetchEnvDef(name string, default_value string) string {
	if value, ok := os.LookupEnv(name); !ok {
		return default_value
	} else {
		return value
	}
}

func render(w io.Writer, name string, data any) {
	log.Printf("Rendering template %s", name)
	if err := rootTemplate.ExecuteTemplate(w, name, data); err != nil {
		log.Fatal(err)
	}
}

type Handler struct {
	db *pgx.Conn
}

var _ http.Handler = &Handler{}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	type User struct {
		Id   int
		Name string
	}

	u := User{}
	err := h.db.QueryRow(context.Background(), "select * from users").Scan(&u.Id, &u.Name)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	render(rw, "hello", struct{ Name string }{Name: u.Name})
}

type AdminHandler struct{}

var _ http.Handler = &AdminHandler{}

func (h *AdminHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(rw, "Admin foo")
}
