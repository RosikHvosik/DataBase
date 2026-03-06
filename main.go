package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB
var tmpl *template.Template

type Product struct {
	ID          int
	Name        string
	Unit        string
	CurrentQty  float64
}

type Supplier struct {
	ID          int
	Name        string
	Phone       string
	BankAccount string
}

type Client struct {
	ID    int
	Name  string
	Phone string
}

type IncomingInvoice struct {
	ID         int
	Number     string
	Date       time.Time
	SupplierID int
	Supplier   string
}

type OutgoingInvoice struct {
	ID       int
	Number   string
	Date     time.Time
	ClientID int
	Client   string
}

type IncomingLine struct {
	ID        int
	InvoiceID int
	ProductID int
	Product   string
	Quantity  float64
	Price     float64
	Total     float64
}

type OutgoingLine struct {
	ID        int
	InvoiceID int
	ProductID int
	Product   string
	Quantity  float64
	Price     float64
	Total     float64
}

type Movement struct {
	Date        time.Time
	Type        string
	InvoiceNum  string
	Quantity    float64
	Price       float64
	Counterpart string
	Balance     float64
}

func initDB() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://appuser:securepass123@localhost:5432/electroshop?sslmode=disable"
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	createTables()
}

func createTables() {
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		name VARCHAR(200) UNIQUE NOT NULL,
		unit VARCHAR(10) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS suppliers (
		id SERIAL PRIMARY KEY,
		name VARCHAR(200) UNIQUE NOT NULL,
		phone VARCHAR(20) NOT NULL,
		bank_account VARCHAR(20) UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS clients (
		id SERIAL PRIMARY KEY,
		name VARCHAR(200) UNIQUE NOT NULL,
		phone VARCHAR(20) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS incoming_invoices (
		id SERIAL PRIMARY KEY,
		number VARCHAR(20) UNIQUE NOT NULL,
		date DATE NOT NULL,
		supplier_id INTEGER REFERENCES suppliers(id) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS outgoing_invoices (
		id SERIAL PRIMARY KEY,
		number VARCHAR(20) UNIQUE NOT NULL,
		date DATE NOT NULL,
		client_id INTEGER REFERENCES clients(id) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS incoming_lines (
		id SERIAL PRIMARY KEY,
		invoice_id INTEGER REFERENCES incoming_invoices(id) ON DELETE CASCADE NOT NULL,
		product_id INTEGER REFERENCES products(id) NOT NULL,
		quantity NUMERIC(10,2) CHECK (quantity > 0) NOT NULL,
		purchase_price NUMERIC(10,2) CHECK (purchase_price > 0) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS outgoing_lines (
		id SERIAL PRIMARY KEY,
		invoice_id INTEGER REFERENCES outgoing_invoices(id) ON DELETE CASCADE NOT NULL,
		product_id INTEGER REFERENCES products(id) NOT NULL,
		quantity NUMERIC(10,2) CHECK (quantity > 0) NOT NULL,
		sale_price NUMERIC(10,2) CHECK (sale_price > 0) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS stock (
		product_id INTEGER PRIMARY KEY REFERENCES products(id),
		current_qty NUMERIC(10,2) DEFAULT 0 NOT NULL
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	defer db.Close()

	tmpl = template.Must(template.ParseGlob("templates/*.html"))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/role", roleHandler)
	http.HandleFunc("/warehouse", warehouseHandler)
	http.HandleFunc("/manager", managerHandler)
	http.HandleFunc("/products", productsHandler)
	http.HandleFunc("/products/add", addProductHandler)
	http.HandleFunc("/products/edit", editProductHandler)
	http.HandleFunc("/products/search", searchProductsHandler)
	http.HandleFunc("/suppliers", suppliersHandler)
	http.HandleFunc("/suppliers/add", addSupplierHandler)
	http.HandleFunc("/suppliers/edit", editSupplierHandler)
	http.HandleFunc("/clients", clientsHandler)
	http.HandleFunc("/clients/add", addClientHandler)
	http.HandleFunc("/clients/edit", editClientHandler)
	http.HandleFunc("/stock", stockHandler)
	http.HandleFunc("/incoming", incomingHandler)
	http.HandleFunc("/incoming/add", addIncomingHandler)
	http.HandleFunc("/incoming/lines", incomingLinesHandler)
	http.HandleFunc("/outgoing", outgoingHandler)
	http.HandleFunc("/outgoing/add", addOutgoingHandler)
	http.HandleFunc("/outgoing/lines", outgoingLinesHandler)
	http.HandleFunc("/reports/movement", movementReportHandler)
	http.HandleFunc("/admin/clear", clearDatabaseHandler)
	http.HandleFunc("/admin/generate", generateDataHandler)

	fmt.Println("Server starting on :8443")
	log.Fatal(http.ListenAndServe(":8443", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "index.html", nil)
}

func roleHandler(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role == "warehouse" {
		http.Redirect(w, r, "/warehouse", http.StatusSeeOther)
	} else if role == "manager" {
		http.Redirect(w, r, "/manager", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func warehouseHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "warehouse.html", nil)
}

func managerHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "manager.html", nil)
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, unit FROM products ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		rows.Scan(&p.ID, &p.Name, &p.Unit)
		products = append(products, p)
	}

	tmpl.ExecuteTemplate(w, "products.html", products)
}

func addProductHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		name := r.FormValue("name")
		unit := r.FormValue("unit")

		var productID int
		err := db.QueryRow("INSERT INTO products (name, unit) VALUES ($1, $2) RETURNING id", name, unit).Scan(&productID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: товар с таким наименованием уже существует"))
			return
		}

		_, err = db.Exec("INSERT INTO stock (product_id, current_qty) VALUES ($1, 0)", productID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при создании записи остатков"))
			return
		}

		w.Header().Set("HX-Redirect", "/products")
		return
	}

	tmpl.ExecuteTemplate(w, "product_form.html", nil)
}

func editProductHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		id := r.FormValue("id")
		name := r.FormValue("name")
		unit := r.FormValue("unit")

		_, err := db.Exec("UPDATE products SET name = $1, unit = $2 WHERE id = $3", name, unit, id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: товар с таким наименованием уже существует"))
			return
		}

		w.Header().Set("HX-Redirect", "/products")
		return
	}

	id := r.URL.Query().Get("id")
	var product Product
	err := db.QueryRow("SELECT id, name, unit FROM products WHERE id = $1", id).Scan(&product.ID, &product.Name, &product.Unit)
	if err != nil {
		http.Error(w, "Товар не найден", http.StatusNotFound)
		return
	}

	tmpl.ExecuteTemplate(w, "product_edit.html", product)
}

func searchProductsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	rows, err := db.Query("SELECT id, name, unit FROM products WHERE name ILIKE $1 ORDER BY name", "%"+query+"%")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		rows.Scan(&p.ID, &p.Name, &p.Unit)
		products = append(products, p)
	}

	tmpl.ExecuteTemplate(w, "products_list.html", products)
}

func suppliersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, phone, bank_account FROM suppliers ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var suppliers []Supplier
	for rows.Next() {
		var s Supplier
		rows.Scan(&s.ID, &s.Name, &s.Phone, &s.BankAccount)
		suppliers = append(suppliers, s)
	}

	tmpl.ExecuteTemplate(w, "suppliers.html", suppliers)
}

func addSupplierHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		name := r.FormValue("name")
		phone := r.FormValue("phone")
		bankAccount := r.FormValue("bank_account")

		_, err := db.Exec("INSERT INTO suppliers (name, phone, bank_account) VALUES ($1, $2, $3)", name, phone, bankAccount)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: поставщик с таким наименованием или расчетным счетом уже существует"))
			return
		}

		w.Header().Set("HX-Redirect", "/suppliers")
		return
	}

	tmpl.ExecuteTemplate(w, "supplier_form.html", nil)
}

func editSupplierHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		id := r.FormValue("id")
		name := r.FormValue("name")
		phone := r.FormValue("phone")
		bankAccount := r.FormValue("bank_account")

		_, err := db.Exec("UPDATE suppliers SET name = $1, phone = $2, bank_account = $3 WHERE id = $4", name, phone, bankAccount, id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: поставщик с таким наименованием или расчетным счетом уже существует"))
			return
		}

		w.Header().Set("HX-Redirect", "/suppliers")
		return
	}

	id := r.URL.Query().Get("id")
	var supplier Supplier
	err := db.QueryRow("SELECT id, name, phone, bank_account FROM suppliers WHERE id = $1", id).Scan(&supplier.ID, &supplier.Name, &supplier.Phone, &supplier.BankAccount)
	if err != nil {
		http.Error(w, "Поставщик не найден", http.StatusNotFound)
		return
	}

	tmpl.ExecuteTemplate(w, "supplier_edit.html", supplier)
}

func clientsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, phone FROM clients ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		rows.Scan(&c.ID, &c.Name, &c.Phone)
		clients = append(clients, c)
	}

	tmpl.ExecuteTemplate(w, "clients.html", clients)
}

func addClientHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		name := r.FormValue("name")
		phone := r.FormValue("phone")

		_, err := db.Exec("INSERT INTO clients (name, phone) VALUES ($1, $2)", name, phone)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: клиент с таким наименованием уже существует"))
			return
		}

		w.Header().Set("HX-Redirect", "/clients")
		return
	}

	tmpl.ExecuteTemplate(w, "client_form.html", nil)
}

func editClientHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		id := r.FormValue("id")
		name := r.FormValue("name")
		phone := r.FormValue("phone")

		_, err := db.Exec("UPDATE clients SET name = $1, phone = $2 WHERE id = $3", name, phone, id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: клиент с таким наименованием уже существует"))
			return
		}

		w.Header().Set("HX-Redirect", "/clients")
		return
	}

	id := r.URL.Query().Get("id")
	var client Client
	err := db.QueryRow("SELECT id, name, phone FROM clients WHERE id = $1", id).Scan(&client.ID, &client.Name, &client.Phone)
	if err != nil {
		http.Error(w, "Клиент не найден", http.StatusNotFound)
		return
	}

	tmpl.ExecuteTemplate(w, "client_edit.html", client)
}

func stockHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT p.id, p.name, p.unit, COALESCE(s.current_qty, 0)
		FROM products p
		LEFT JOIN stock s ON p.id = s.product_id
		ORDER BY p.name
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		rows.Scan(&p.ID, &p.Name, &p.Unit, &p.CurrentQty)
		products = append(products, p)
	}

	tmpl.ExecuteTemplate(w, "stock.html", products)
}

func incomingHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT i.id, i.number, i.date, i.supplier_id, s.name
		FROM incoming_invoices i
		JOIN suppliers s ON i.supplier_id = s.id
		ORDER BY i.date DESC, i.number DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var invoices []IncomingInvoice
	for rows.Next() {
		var inv IncomingInvoice
		rows.Scan(&inv.ID, &inv.Number, &inv.Date, &inv.SupplierID, &inv.Supplier)
		invoices = append(invoices, inv)
	}

	tmpl.ExecuteTemplate(w, "incoming.html", invoices)
}

func addIncomingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		number := r.FormValue("number")
		date := r.FormValue("date")
		supplierID := r.FormValue("supplier_id")

		tx, err := db.Begin()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при создании транзакции"))
			return
		}

		var invoiceID int
		err = tx.QueryRow("INSERT INTO incoming_invoices (number, date, supplier_id) VALUES ($1, $2, $3) RETURNING id", number, date, supplierID).Scan(&invoiceID)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: накладная с таким номером уже существует"))
			return
		}

		productIDs := r.Form["product_id"]
		quantities := r.Form["quantity"]
		prices := r.Form["price"]

		if len(productIDs) == 0 {
			tx.Rollback()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: необходимо добавить хотя бы одну строку товара"))
			return
		}

		for i := range productIDs {
			productID := productIDs[i]
			quantity := quantities[i]
			price := prices[i]

			_, err = tx.Exec("INSERT INTO incoming_lines (invoice_id, product_id, quantity, purchase_price) VALUES ($1, $2, $3, $4)", invoiceID, productID, quantity, price)
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка при добавлении строки накладной"))
				return
			}

			_, err = tx.Exec(`
				INSERT INTO stock (product_id, current_qty) VALUES ($1, $2)
				ON CONFLICT (product_id) DO UPDATE SET current_qty = stock.current_qty + $2
			`, productID, quantity)
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка при обновлении остатков"))
				return
			}
		}

		tx.Commit()
		w.Header().Set("HX-Redirect", "/incoming")
		return
	}

	suppliers, _ := db.Query("SELECT id, name FROM suppliers ORDER BY name")
	defer suppliers.Close()
	var supplierList []Supplier
	for suppliers.Next() {
		var s Supplier
		suppliers.Scan(&s.ID, &s.Name)
		supplierList = append(supplierList, s)
	}

	products, _ := db.Query("SELECT id, name, unit FROM products ORDER BY name")
	defer products.Close()
	var productList []Product
	for products.Next() {
		var p Product
		products.Scan(&p.ID, &p.Name, &p.Unit)
		productList = append(productList, p)
	}

	data := map[string]interface{}{
		"Suppliers": supplierList,
		"Products":  productList,
	}

	tmpl.ExecuteTemplate(w, "incoming_form.html", data)
}

func incomingLinesHandler(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.URL.Query().Get("id")

	rows, err := db.Query(`
		SELECT l.id, l.invoice_id, l.product_id, p.name, l.quantity, l.purchase_price, l.quantity * l.purchase_price
		FROM incoming_lines l
		JOIN products p ON l.product_id = p.id
		WHERE l.invoice_id = $1
	`, invoiceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lines []IncomingLine
	for rows.Next() {
		var line IncomingLine
		rows.Scan(&line.ID, &line.InvoiceID, &line.ProductID, &line.Product, &line.Quantity, &line.Price, &line.Total)
		lines = append(lines, line)
	}

	tmpl.ExecuteTemplate(w, "incoming_lines.html", lines)
}

func outgoingHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT o.id, o.number, o.date, o.client_id, c.name
		FROM outgoing_invoices o
		JOIN clients c ON o.client_id = c.id
		ORDER BY o.date DESC, o.number DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var invoices []OutgoingInvoice
	for rows.Next() {
		var inv OutgoingInvoice
		rows.Scan(&inv.ID, &inv.Number, &inv.Date, &inv.ClientID, &inv.Client)
		invoices = append(invoices, inv)
	}

	tmpl.ExecuteTemplate(w, "outgoing.html", invoices)
}

func addOutgoingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		number := r.FormValue("number")
		date := r.FormValue("date")
		clientID := r.FormValue("client_id")

		tx, err := db.Begin()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при создании транзакции"))
			return
		}

		var invoiceID int
		err = tx.QueryRow("INSERT INTO outgoing_invoices (number, date, client_id) VALUES ($1, $2, $3) RETURNING id", number, date, clientID).Scan(&invoiceID)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: накладная с таким номером уже существует"))
			return
		}

		productIDs := r.Form["product_id"]
		quantities := r.Form["quantity"]
		prices := r.Form["price"]

		if len(productIDs) == 0 {
			tx.Rollback()
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Ошибка: необходимо добавить хотя бы одну строку товара"))
			return
		}

		for i := range productIDs {
			productID := productIDs[i]
			quantity := quantities[i]
			price := prices[i]

			var currentQty float64
			err = tx.QueryRow("SELECT current_qty FROM stock WHERE product_id = $1", productID).Scan(&currentQty)
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Ошибка: товар не найден на складе"))
				return
			}

			qtyFloat, _ := strconv.ParseFloat(quantity, 64)
			if currentQty < qtyFloat {
				tx.Rollback()
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Ошибка: недостаточно товара на складе. Доступно: " + fmt.Sprintf("%.2f", currentQty)))
				return
			}

			_, err = tx.Exec("INSERT INTO outgoing_lines (invoice_id, product_id, quantity, sale_price) VALUES ($1, $2, $3, $4)", invoiceID, productID, quantity, price)
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка при добавлении строки накладной"))
				return
			}

			_, err = tx.Exec("UPDATE stock SET current_qty = current_qty - $1 WHERE product_id = $2", quantity, productID)
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка при обновлении остатков"))
				return
			}
		}

		tx.Commit()
		w.Header().Set("HX-Redirect", "/outgoing")
		return
	}

	clients, _ := db.Query("SELECT id, name FROM clients ORDER BY name")
	defer clients.Close()
	var clientList []Client
	for clients.Next() {
		var c Client
		clients.Scan(&c.ID, &c.Name)
		clientList = append(clientList, c)
	}

	products, _ := db.Query(`
		SELECT p.id, p.name, p.unit, COALESCE(s.current_qty, 0)
		FROM products p
		LEFT JOIN stock s ON p.id = s.product_id
		WHERE COALESCE(s.current_qty, 0) > 0
		ORDER BY p.name
	`)
	defer products.Close()
	var productList []Product
	for products.Next() {
		var p Product
		products.Scan(&p.ID, &p.Name, &p.Unit, &p.CurrentQty)
		productList = append(productList, p)
	}

	data := map[string]interface{}{
		"Clients":  clientList,
		"Products": productList,
	}

	tmpl.ExecuteTemplate(w, "outgoing_form.html", data)
}

func outgoingLinesHandler(w http.ResponseWriter, r *http.Request) {
	invoiceID := r.URL.Query().Get("id")

	rows, err := db.Query(`
		SELECT l.id, l.invoice_id, l.product_id, p.name, l.quantity, l.sale_price, l.quantity * l.sale_price
		FROM outgoing_lines l
		JOIN products p ON l.product_id = p.id
		WHERE l.invoice_id = $1
	`, invoiceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lines []OutgoingLine
	for rows.Next() {
		var line OutgoingLine
		rows.Scan(&line.ID, &line.InvoiceID, &line.ProductID, &line.Product, &line.Quantity, &line.Price, &line.Total)
		lines = append(lines, line)
	}

	tmpl.ExecuteTemplate(w, "outgoing_lines.html", lines)
}

func movementReportHandler(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("product_id")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	if productID == "" {
		products, _ := db.Query("SELECT id, name FROM products ORDER BY name")
		defer products.Close()
		var productList []Product
		for products.Next() {
			var p Product
			products.Scan(&p.ID, &p.Name)
			productList = append(productList, p)
		}

		tmpl.ExecuteTemplate(w, "movement_form.html", productList)
		return
	}

	var movements []Movement
	var balance float64

	incoming, _ := db.Query(`
		SELECT i.date, i.number, l.quantity, l.purchase_price, s.name
		FROM incoming_lines l
		JOIN incoming_invoices i ON l.invoice_id = i.id
		JOIN suppliers s ON i.supplier_id = s.id
		WHERE l.product_id = $1 AND i.date BETWEEN $2 AND $3
		ORDER BY i.date, i.number
	`, productID, dateFrom, dateTo)
	defer incoming.Close()

	for incoming.Next() {
		var m Movement
		incoming.Scan(&m.Date, &m.InvoiceNum, &m.Quantity, &m.Price, &m.Counterpart)
		m.Type = "Приход"
		balance += m.Quantity
		m.Balance = balance
		movements = append(movements, m)
	}

	outgoing, _ := db.Query(`
		SELECT o.date, o.number, l.quantity, l.sale_price, c.name
		FROM outgoing_lines l
		JOIN outgoing_invoices o ON l.invoice_id = o.id
		JOIN clients c ON o.client_id = c.id
		WHERE l.product_id = $1 AND o.date BETWEEN $2 AND $3
		ORDER BY o.date, o.number
	`, productID, dateFrom, dateTo)
	defer outgoing.Close()

	for outgoing.Next() {
		var m Movement
		outgoing.Scan(&m.Date, &m.InvoiceNum, &m.Quantity, &m.Price, &m.Counterpart)
		m.Type = "Расход"
		m.Quantity = -m.Quantity
		balance += m.Quantity
		m.Balance = balance
		movements = append(movements, m)
	}

	var productName string
	db.QueryRow("SELECT name FROM products WHERE id = $1", productID).Scan(&productName)

	data := map[string]interface{}{
		"ProductName": productName,
		"Movements":   movements,
		"DateFrom":    dateFrom,
		"DateTo":      dateTo,
	}

	tmpl.ExecuteTemplate(w, "movement_report.html", data)
}

func clearDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Ошибка при создании транзакции"))
		return
	}

	_, err = tx.Exec("TRUNCATE TABLE outgoing_lines, incoming_lines, outgoing_invoices, incoming_invoices, stock, products, suppliers, clients RESTART IDENTITY CASCADE")
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Ошибка при очистке базы данных"))
		return
	}

	tx.Commit()
	w.Header().Set("HX-Redirect", "/manager")
}

func generateDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Ошибка при создании транзакции"))
		return
	}

	productNames := []string{
		"Молоко пастеризованное 3.2%", "Хлеб Бородинский", "Масло сливочное 82.5%", "Сыр Российский",
		"Творог 9%", "Кефир 2.5%", "Сметана 20%", "Йогурт натуральный", "Яйца куриные С1",
		"Мука пшеничная высший сорт", "Сахар-песок", "Соль поваренная", "Рис круглозерный",
		"Гречка ядрица", "Макароны спагетти", "Масло подсолнечное", "Чай черный", "Кофе растворимый",
		"Печенье сахарное", "Конфеты шоколадные", "Шоколад молочный", "Вафли", "Пряники",
		"Колбаса вареная Докторская", "Сосиски молочные", "Ветчина", "Курица охлажденная",
		"Свинина", "Говядина", "Рыба замороженная", "Картофель", "Морковь", "Лук репчатый",
		"Капуста белокочанная", "Помидоры", "Огурцы", "Перец болгарский", "Яблоки",
		"Бананы", "Апельсины", "Мандарины", "Виноград", "Груши", "Сок яблочный",
		"Сок апельсиновый", "Вода минеральная", "Лимонад", "Квас", "Пиво светлое",
		"Майонез", "Кетчуп", "Горчица", "Уксус", "Специи", "Лавровый лист",
		"Перец черный молотый", "Соус соевый", "Томатная паста", "Консервы рыбные",
		"Консервы мясные", "Горошек зеленый", "Кукуруза консервированная", "Фасоль",
		"Оливки", "Маслины", "Варенье клубничное", "Джем абрикосовый", "Мед натуральный",
		"Орехи грецкие", "Семечки подсолнечные", "Изюм", "Курага", "Чернослив",
		"Сухари панировочные", "Дрожжи", "Разрыхлитель", "Ванилин", "Желатин",
		"Крахмал картофельный", "Какао-порошок", "Сгущенное молоко", "Сливки 20%",
		"Мороженое пломбир", "Пельмени", "Вареники", "Блины", "Пицца замороженная",
		"Котлеты", "Наггетсы куриные", "Рыбные палочки", "Креветки", "Кальмары",
		"Крабовые палочки", "Икра красная", "Сельдь соленая", "Скумбрия копченая",
		"Салат Оливье", "Салат Цезарь", "Торт Наполеон", "Пирожное Эклер",
		"Зефир", "Мармелад", "Халва", "Козинаки", "Батон нарезной",
	}

	units := []string{"шт.", "кг", "л", "уп"}
	
	for i := 0; i < 100; i++ {
		var productID int
		productName := productNames[i%len(productNames)]
		if i >= len(productNames) {
			productName = fmt.Sprintf("%s %d", productName, i/len(productNames)+1)
		}
		
		err = tx.QueryRow("INSERT INTO products (name, unit) VALUES ($1, $2) RETURNING id",
			productName, units[i%4]).Scan(&productID)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при генерации товаров"))
			return
		}
		_, err = tx.Exec("INSERT INTO stock (product_id, current_qty) VALUES ($1, 0)", productID)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при генерации остатков"))
			return
		}
	}

	for i := 1; i <= 100; i++ {
		_, err = tx.Exec("INSERT INTO suppliers (name, phone, bank_account) VALUES ($1, $2, $3)",
			fmt.Sprintf("Поставщик %d", i),
			fmt.Sprintf("+7%010d", 9000000000+i),
			fmt.Sprintf("4070281000000000%04d", i))
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при генерации поставщиков"))
			return
		}
	}

	for i := 1; i <= 100; i++ {
		_, err = tx.Exec("INSERT INTO clients (name, phone) VALUES ($1, $2)",
			fmt.Sprintf("Клиент %d", i),
			fmt.Sprintf("+7%010d", 9100000000+i))
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при генерации клиентов"))
			return
		}
	}

	for i := 1; i <= 100; i++ {
		var invoiceID int
		err = tx.QueryRow("INSERT INTO incoming_invoices (number, date, supplier_id) VALUES ($1, $2, $3) RETURNING id",
			fmt.Sprintf("ПН-%03d", i),
			time.Now().AddDate(0, 0, -i),
			(i%100)+1).Scan(&invoiceID)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при генерации приходных накладных"))
			return
		}

		for j := 1; j <= 3; j++ {
			productID := ((i-1)*3 + j)
			if productID > 100 {
				productID = productID % 100
			}
			if productID == 0 {
				productID = 1
			}
			quantity := float64(j * 10)
			_, err = tx.Exec("INSERT INTO incoming_lines (invoice_id, product_id, quantity, purchase_price) VALUES ($1, $2, $3, $4)",
				invoiceID, productID, quantity, float64(50+j*10))
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка при генерации строк приходных накладных"))
				return
			}

			_, err = tx.Exec("UPDATE stock SET current_qty = current_qty + $1 WHERE product_id = $2", quantity, productID)
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка при обновлении остатков"))
				return
			}
		}
	}

	for i := 1; i <= 100; i++ {
		var invoiceID int
		err = tx.QueryRow("INSERT INTO outgoing_invoices (number, date, client_id) VALUES ($1, $2, $3) RETURNING id",
			fmt.Sprintf("РН-%03d", i),
			time.Now().AddDate(0, 0, -i+50),
			(i%100)+1).Scan(&invoiceID)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Ошибка при генерации расходных накладных"))
			return
		}

		for j := 1; j <= 2; j++ {
			productID := ((i-1)*2 + j)
			if productID > 100 {
				productID = productID % 100
			}
			if productID == 0 {
				productID = 1
			}

			var currentQty float64
			err = tx.QueryRow("SELECT current_qty FROM stock WHERE product_id = $1", productID).Scan(&currentQty)
			if err != nil || currentQty < float64(j*5) {
				continue
			}

			quantity := float64(j * 5)
			_, err = tx.Exec("INSERT INTO outgoing_lines (invoice_id, product_id, quantity, sale_price) VALUES ($1, $2, $3, $4)",
				invoiceID, productID, quantity, float64(100+j*20))
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка при генерации строк расходных накладных"))
				return
			}

			_, err = tx.Exec("UPDATE stock SET current_qty = current_qty - $1 WHERE product_id = $2", quantity, productID)
			if err != nil {
				tx.Rollback()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Ошибка при обновлении остатков"))
				return
			}
		}
	}

	tx.Commit()
	w.Header().Set("HX-Redirect", "/manager")
}
