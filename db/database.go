package db

import (
	"database/sql"
	"nexus-hub-core/api"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	Conn *sql.DB
}

// Структуры данных для JSON
type PrintJob struct {
	ID              int64     `json:"id"`
	Filename        string    `json:"filename"`
	Filament        string    `json:"filament_type"`
	PrinterModel    string    `json:"printer_model"`
	EstimatedTimeS  uint64    `json:"estimated_time_s"`
	Status          string    `json:"status"`
	TargetClusterID int64     `json:"target_cluster_id"`
	TargetPrinterID int64     `json:"target_printer_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type Cluster struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	TargetModel string    `json:"target_model"`
	IsEnabled   bool      `json:"is_enabled"`
	Printers    []Printer `json:"printers"`
}

type Printer struct {
	ID        int64  `json:"id"`
	ClusterID int64  `json:"cluster_id"`
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
	Status    string `json:"status"`
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	d := &Database{Conn: db}
	err = d.setupTables()
	return d, err
}

func (d *Database) setupTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS system_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			module TEXT NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS host_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			cpu_load REAL NOT NULL,
			gpu_load REAL NOT NULL,
			ram_usage_mb INTEGER NOT NULL,
			gpu_temp REAL,
			cpu_temp REAL
		)`,
		`CREATE TABLE IF NOT EXISTS clusters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			target_model TEXT UNIQUE NOT NULL,
			is_enabled INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS printers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			ip_address TEXT NOT NULL,
			status TEXT DEFAULT 'IDLE',
			FOREIGN KEY(cluster_id) REFERENCES clusters(id)
		)`,
		`CREATE TABLE IF NOT EXISTS print_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			filament_type TEXT,
			printer_model TEXT NOT NULL,
			estimated_time_s INTEGER,
			status TEXT NOT NULL,
			target_cluster_id INTEGER DEFAULT 0,
			target_printer_id INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL
		)`,
	}
	for _, q := range queries {
		if _, err := d.Conn.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// --- Методы работы с задачами (Jobs) ---

func (d *Database) AddPrintJob(filename, filament, printerModel string, timeS uint64) error {
	_, err := d.Conn.Exec(
		"INSERT INTO print_jobs (filename, filament_type, printer_model, estimated_time_s, status, created_at) VALUES (?, ?, ?, ?, 'PENDING', ?)",
		filename, filament, printerModel, timeS, time.Now(),
	)
	return err
}

func (d *Database) GetPendingJobs() ([]PrintJob, error) {
	rows, err := d.Conn.Query("SELECT id, filename, filament_type, printer_model, estimated_time_s, status, target_cluster_id, target_printer_id, created_at FROM print_jobs WHERE status = 'PENDING' ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []PrintJob{}
	for rows.Next() {
		var j PrintJob
		if err := rows.Scan(&j.ID, &j.Filename, &j.Filament, &j.PrinterModel, &j.EstimatedTimeS, &j.Status, &j.TargetClusterID, &j.TargetPrinterID, &j.CreatedAt); err == nil {
			jobs = append(jobs, j)
		}
	}
	return jobs, nil
}

func (d *Database) GetJobList() ([]PrintJob, error) {
	rows, err := d.Conn.Query("SELECT id, filename, filament_type, printer_model, estimated_time_s, status, target_cluster_id, target_printer_id, created_at FROM print_jobs WHERE status IN ('ASSIGNED', 'PRINTING') ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []PrintJob{}
	for rows.Next() {
		var j PrintJob
		if err := rows.Scan(&j.ID, &j.Filename, &j.Filament, &j.PrinterModel, &j.EstimatedTimeS, &j.Status, &j.TargetClusterID, &j.TargetPrinterID, &j.CreatedAt); err == nil {
			jobs = append(jobs, j)
		}
	}
	return jobs, nil
}

func (d *Database) AssignJob(jobID, clusterID, printerID int64) error {
	_, err := d.Conn.Exec("UPDATE print_jobs SET target_cluster_id = ?, target_printer_id = ?, status = 'ASSIGNED' WHERE id = ?", clusterID, printerID, jobID)
	return err
}

func (d *Database) GetJobByID(id int64) (PrintJob, error) {
	var j PrintJob
	err := d.Conn.QueryRow("SELECT id, filename, filament_type, printer_model, estimated_time_s, status, target_cluster_id, target_printer_id, created_at FROM print_jobs WHERE id = ?", id).
		Scan(&j.ID, &j.Filename, &j.Filament, &j.PrinterModel, &j.EstimatedTimeS, &j.Status, &j.TargetClusterID, &j.TargetPrinterID, &j.CreatedAt)
	return j, err
}

func (d *Database) UpdateJobStatus(id int64, status string) error {
	_, err := d.Conn.Exec("UPDATE print_jobs SET status = ? WHERE id = ?", status, id)
	return err
}

func (d *Database) GetNextPendingJob() (int64, string, string, error) {
	var id int64
	var filename, filament string
	err := d.Conn.QueryRow("SELECT id, filename, filament_type FROM print_jobs WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT 1").Scan(&id, &filename, &filament)
	if err == sql.ErrNoRows {
		return 0, "", "", nil
	}
	return id, filename, filament, err
}

// --- Методы работы с Кластерами и Принтерами ---

func (d *Database) CreateCluster(name, targetModel string) error {
	_, err := d.Conn.Exec("INSERT INTO clusters (name, target_model) VALUES (?, ?)", name, targetModel)
	return err
}

func (d *Database) ToggleCluster(id int64, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := d.Conn.Exec("UPDATE clusters SET is_enabled = ? WHERE id = ?", val, id)
	return err
}

func (d *Database) GetClustersWithPrinters() ([]Cluster, error) {
	rows, err := d.Conn.Query("SELECT id, name, target_model, is_enabled FROM clusters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clusters := []Cluster{}
	for rows.Next() {
		var c Cluster
		var enabled int
		rows.Scan(&c.ID, &c.Name, &c.TargetModel, &enabled)
		c.IsEnabled = enabled == 1
		c.Printers = []Printer{}
		clusters = append(clusters, c)
	}

	for i := range clusters {
		pRows, _ := d.Conn.Query("SELECT id, cluster_id, name, ip_address, status FROM printers WHERE cluster_id = ?", clusters[i].ID)
		for pRows.Next() {
			var p Printer
			pRows.Scan(&p.ID, &p.ClusterID, &p.Name, &p.IPAddress, &p.Status)
			clusters[i].Printers = append(clusters[i].Printers, p)
		}
		pRows.Close()
	}
	return clusters, nil
}

func (d *Database) AddPrinter(clusterID int64, name, ip string) error {
	_, err := d.Conn.Exec("INSERT INTO printers (cluster_id, name, ip_address, status) VALUES (?, ?, ?, 'IDLE')", clusterID, name, ip)
	return err
}

func (d *Database) UpdatePrinterStatus(id int64, status string) error {
	_, err := d.Conn.Exec("UPDATE printers SET status = ? WHERE id = ?", status, id)
	return err
}

func (d *Database) DeleteCluster(id int64) error {
	tx, err := d.Conn.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM printers WHERE cluster_id = ?", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM clusters WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *Database) GetUnassignedModels() ([]string, error) {
	query := `SELECT DISTINCT printer_model FROM print_jobs 
			  WHERE printer_model NOT IN (SELECT target_model FROM clusters)`
	rows, err := d.Conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	models := []string{}
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err == nil {
			models = append(models, m)
		}
	}
	return models, nil
}

// --- Служебные методы (Логи и Метрики) ---

func (d *Database) SaveEvent(event api.NexusPacket) error {
	now := time.Now()
	if event.EventType == "HOST_METRICS" {
		m := event.Payload.(api.HostHardwareState)
		_, err := d.Conn.Exec(
			"INSERT INTO host_metrics (timestamp, cpu_load, gpu_load, ram_usage_mb, gpu_temp, cpu_temp) VALUES (?, ?, ?, ?, ?, ?)",
			now, m.CPUUsageTotal, m.GPULoadPercent, m.RAMUsedMB, m.GPUTempCelsius, m.CPUPackageTemp,
		)
		return err
	} else if event.Log != nil {
		_, err := d.Conn.Exec(
			"INSERT INTO system_logs (timestamp, module, level, message) VALUES (?, ?, ?, ?)",
			event.Log.Timestamp, event.Log.Module, event.Log.Level, event.Log.Message,
		)
		return err
	}
	return nil
}
