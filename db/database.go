package db

import (
	"database/sql"
	"log"
	"nexus-hub-core/api"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	Conn *sql.DB
}

// PrintJob represents a print job in the queue
type PrintJob struct {
	ID              int64      `json:"id"`
	Filename        string     `json:"filename"`
	Filament        string     `json:"filament_type"`
	PrinterModel    string     `json:"printer_model"`
	EstimatedTimeS  uint64     `json:"estimated_time_s"`
	FilamentWeight  float32    `json:"filament_weight_g"`
	ThumbnailURL    string     `json:"thumbnail_url"`
	Status          string     `json:"status"`
	TargetClusterID int64      `json:"target_cluster_id"`
	TargetPrinterID int64      `json:"target_printer_id"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

// PrintHistory represents a completed print job for analytics
type PrintHistory struct {
	ID             int64     `json:"id"`
	JobID          int64     `json:"job_id"`
	Filename       string    `json:"filename"`
	Filament       string    `json:"filament_type"`
	PrinterModel   string    `json:"printer_model"`
	PrinterID      int64     `json:"printer_id"`
	PrinterName    string    `json:"printer_name"`
	ClusterID      int64     `json:"cluster_id"`
	ClusterName    string    `json:"cluster_name"`
	EstimatedTimeS uint64    `json:"estimated_time_s"`
	ActualTimeS    uint64    `json:"actual_time_s"`
	FilamentUsedM  float32   `json:"filament_used_m"`
	Status         string    `json:"status"` // COMPLETED, FAILED, CANCELLED
	FailureReason  string    `json:"failure_reason,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
}

// PrinterWearStats represents wear-level statistics for a printer
type PrinterWearStats struct {
	PrinterID        int64   `json:"printer_id"`
	PrinterName      string  `json:"printer_name"`
	TotalPrintTimeS  int64   `json:"total_print_time_s"`
	TotalPrints      int64   `json:"total_prints"`
	SuccessfulPrints int64   `json:"successful_prints"`
	FailedPrints     int64   `json:"failed_prints"`
	SuccessRate      float32 `json:"success_rate"`
	AvgPrintTimeS    float64 `json:"avg_print_time_s"`
	WearScore        float64 `json:"wear_score"` // Lower is better (less wear)
}

// Cluster represents a group of printers
type Cluster struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	TargetModel string    `json:"target_model"`
	IsEnabled   bool      `json:"is_enabled"`
	Printers    []Printer `json:"printers"`
}

// Printer represents a 3D printer
type Printer struct {
	ID          int64      `json:"id"`
	ClusterID   int64      `json:"cluster_id"`
	Name        string     `json:"name"`
	IPAddress   string     `json:"ip_address"`
	Status      string     `json:"status"`
	PrinterType string     `json:"printer_type"`          // MOONRAKER, BAMBULAB, etc.
	Serial      string     `json:"serial,omitempty"`      // Bambu serial number
	AccessCode  string     `json:"access_code,omitempty"` // Bambu LAN access code
	Username    string     `json:"username,omitempty"`
	Password    string     `json:"password,omitempty"`
	AccessToken string     `json:"access_token,omitempty"`
	Secret      string     `json:"secret,omitempty"`
	Extra       string     `json:"extra,omitempty"` // JSON-encoded vendor-specific config
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

// FilamentSpool represents a filament spool in inventory
type FilamentSpool struct {
	ID          int64   `json:"id"`
	Material    string  `json:"material"` // PLA, PETG, ABS, etc.
	Color       string  `json:"color"`    // Color name
	ColorHex    string  `json:"color_hex"`
	TotalWeight float32 `json:"total_weight_g"`
	Remaining   float32 `json:"remaining_weight_g"`
	AssignedTo  int64   `json:"assigned_printer_id"`
	PrinterName string  `json:"printer_name,omitempty"`
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Configure connection pooling
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Enable WAL mode and set busy timeout for concurrency
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA busy_timeout=5000;")

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
			printer_type TEXT DEFAULT 'MOONRAKER',
			serial TEXT,
			access_code TEXT,
			username TEXT,
			password TEXT,
			access_token TEXT,
			secret TEXT,
			extra TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
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
			created_at DATETIME NOT NULL,
			started_at DATETIME,
			completed_at DATETIME,
			thumbnail_url TEXT,
			filament_weight_g REAL
		)`,
		`CREATE TABLE IF NOT EXISTS spools (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			material TEXT NOT NULL,
			color TEXT NOT NULL,
			color_hex TEXT DEFAULT '#FFFFFF',
			total_weight_g REAL NOT NULL,
			remaining_weight_g REAL NOT NULL,
			assigned_printer_id INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(assigned_printer_id) REFERENCES printers(id)
		)`,
		`CREATE TABLE IF NOT EXISTS print_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER,
			filename TEXT NOT NULL,
			filament_type TEXT,
			printer_model TEXT NOT NULL,
			printer_id INTEGER NOT NULL,
			printer_name TEXT,
			cluster_id INTEGER,
			cluster_name TEXT,
			estimated_time_s INTEGER,
			actual_time_s INTEGER,
			filament_used_m REAL,
			status TEXT NOT NULL,
			failure_reason TEXT,
			started_at DATETIME NOT NULL,
			completed_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_printer ON print_history(printer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_history_status ON print_history(status)`,
		`CREATE INDEX IF NOT EXISTS idx_history_date ON print_history(completed_at)`,
		`CREATE TABLE IF NOT EXISTS printer_stats (
			printer_id INTEGER PRIMARY KEY,
			total_print_time_s INTEGER DEFAULT 0,
			total_prints INTEGER DEFAULT 0,
			successful_prints INTEGER DEFAULT 0,
			failed_prints INTEGER DEFAULT 0,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := d.Conn.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// --- Print Job Methods ---

func (d *Database) AddPrintJob(filename, filament, printerModel string, timeS uint64) error {
	_, err := d.Conn.Exec(
		"INSERT INTO print_jobs (filename, filament_type, printer_model, estimated_time_s, status, created_at) VALUES (?, ?, ?, ?, 'PENDING', ?)",
		filename, filament, printerModel, timeS, time.Now(),
	)
	return err
}

func (d *Database) GetPendingJobs() ([]PrintJob, error) {
	rows, err := d.Conn.Query(`
		SELECT id, filename, filament_type, printer_model, estimated_time_s, filament_weight_g, thumbnail_url, status,
		       target_cluster_id, target_printer_id, created_at, started_at, completed_at
		FROM print_jobs
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []PrintJob{}
	for rows.Next() {
		var j PrintJob
		if err := rows.Scan(&j.ID, &j.Filename, &j.Filament, &j.PrinterModel,
			&j.EstimatedTimeS, &j.FilamentWeight, &j.ThumbnailURL, &j.Status,
			&j.TargetClusterID, &j.TargetPrinterID, &j.CreatedAt, &j.StartedAt, &j.CompletedAt); err == nil {
			jobs = append(jobs, j)
		}
	}
	return jobs, nil
}

func (d *Database) GetJobList() ([]PrintJob, error) {
	rows, err := d.Conn.Query(`
		SELECT id, filename, filament_type, printer_model, estimated_time_s, filament_weight_g, thumbnail_url, status,
		       target_cluster_id, target_printer_id, created_at, started_at, completed_at
		FROM print_jobs
		WHERE status IN ('ASSIGNED', 'PRINTING')
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []PrintJob{}
	for rows.Next() {
		var j PrintJob
		if err := rows.Scan(&j.ID, &j.Filename, &j.Filament, &j.PrinterModel,
			&j.EstimatedTimeS, &j.FilamentWeight, &j.ThumbnailURL, &j.Status,
			&j.TargetClusterID, &j.TargetPrinterID, &j.CreatedAt, &j.StartedAt, &j.CompletedAt); err == nil {
			jobs = append(jobs, j)
		}
	}
	return jobs, nil
}

func (d *Database) AssignJob(jobID, clusterID, printerID int64) error {
	now := time.Now()
	_, err := d.Conn.Exec(`
		UPDATE print_jobs 
		SET target_cluster_id = ?, target_printer_id = ?, status = 'PRINTING', started_at = ? 
		WHERE id = ?
	`, clusterID, printerID, now, jobID)
	return err
}

func (d *Database) GetJobByID(id int64) (PrintJob, error) {
	var j PrintJob
	err := d.Conn.QueryRow(`
		SELECT id, filename, filament_type, printer_model, estimated_time_s, status, 
		       target_cluster_id, target_printer_id, created_at, started_at, completed_at 
		FROM print_jobs WHERE id = ?
	`, id).Scan(&j.ID, &j.Filename, &j.Filament, &j.PrinterModel, &j.EstimatedTimeS,
		&j.Status, &j.TargetClusterID, &j.TargetPrinterID, &j.CreatedAt, &j.StartedAt, &j.CompletedAt)
	return j, err
}

func (d *Database) UpdateJobStatus(id int64, status string) error {
	if status == "COMPLETED" || status == "FAILED" || status == "CANCELLED" {
		now := time.Now()
		_, err := d.Conn.Exec(`
			UPDATE print_jobs 
			SET status = ?, completed_at = ? 
			WHERE id = ?
		`, status, now, id)

		// Also archive to history
		if err == nil {
			d.archiveJobToHistory(id, status)
		}
		return err
	}

	_, err := d.Conn.Exec("UPDATE print_jobs SET status = ? WHERE id = ?", status, id)
	return err
}

func (d *Database) archiveJobToHistory(jobID int64, finalStatus string) {
	var job PrintJob
	err := d.Conn.QueryRow(`
		SELECT id, filename, filament_type, printer_model, estimated_time_s, status, 
		       target_cluster_id, target_printer_id, created_at, started_at, completed_at 
		FROM print_jobs WHERE id = ?
	`, jobID).Scan(&job.ID, &job.Filename, &job.Filament, &job.PrinterModel,
		&job.EstimatedTimeS, &job.Status, &job.TargetClusterID, &job.TargetPrinterID,
		&job.CreatedAt, &job.StartedAt, &job.CompletedAt)

	if err != nil {
		return
	}

	// Get printer name
	var printerName, clusterName string
	d.Conn.QueryRow("SELECT name FROM printers WHERE id = ?", job.TargetPrinterID).Scan(&printerName)
	d.Conn.QueryRow("SELECT name FROM clusters WHERE id = ?", job.TargetClusterID).Scan(&clusterName)

	// Calculate actual print time
	actualTime := uint64(0)
	if job.StartedAt != nil && job.CompletedAt != nil {
		actualTime = uint64(job.CompletedAt.Sub(*job.StartedAt).Seconds())
	}

	_, err = d.Conn.Exec(`
		INSERT INTO print_history 
		(job_id, filename, filament_type, printer_model, printer_id, printer_name, 
		 cluster_id, cluster_name, estimated_time_s, actual_time_s, status, 
		 started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.Filename, job.Filament, job.PrinterModel, job.TargetPrinterID,
		printerName, job.TargetClusterID, clusterName, job.EstimatedTimeS, actualTime,
		finalStatus, job.StartedAt, job.CompletedAt)

	if err == nil {
		// Update printer stats
		d.updatePrinterStats(job.TargetPrinterID, actualTime, finalStatus == "COMPLETED")
	}
}

func (d *Database) updatePrinterStats(printerID int64, printTimeS uint64, success bool) {
	// Check if record exists
	var exists int
	d.Conn.QueryRow("SELECT COUNT(*) FROM printer_stats WHERE printer_id = ?", printerID).Scan(&exists)

	if exists == 0 {
		d.Conn.Exec(`
			INSERT INTO printer_stats 
			(printer_id, total_print_time_s, total_prints, successful_prints, failed_prints)
			VALUES (?, ?, 1, ?, ?)
		`, printerID, printTimeS, map[bool]int{true: 1, false: 0}[success], map[bool]int{true: 0, false: 1}[success])
	} else {
		if success {
			d.Conn.Exec(`
				UPDATE printer_stats 
				SET total_print_time_s = total_print_time_s + ?, 
				    total_prints = total_prints + 1,
				    successful_prints = successful_prints + 1,
				    last_updated = CURRENT_TIMESTAMP
				WHERE printer_id = ?
			`, printTimeS, printerID)
		} else {
			d.Conn.Exec(`
				UPDATE printer_stats 
				SET total_print_time_s = total_print_time_s + ?, 
				    total_prints = total_prints + 1,
				    failed_prints = failed_prints + 1,
				    last_updated = CURRENT_TIMESTAMP
				WHERE printer_id = ?
			`, printTimeS, printerID)
		}
	}
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

// --- Print History Methods ---

func (d *Database) GetPrintHistory(limit int) ([]PrintHistory, error) {
	rows, err := d.Conn.Query(`
		SELECT id, job_id, filename, filament_type, printer_model, printer_id, 
		       printer_name, cluster_id, cluster_name, estimated_time_s, actual_time_s,
		       filament_used_m, status, failure_reason, started_at, completed_at
		FROM print_history
		ORDER BY completed_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []PrintHistory{}
	for rows.Next() {
		var h PrintHistory
		if err := rows.Scan(&h.ID, &h.JobID, &h.Filename, &h.Filament, &h.PrinterModel,
			&h.PrinterID, &h.PrinterName, &h.ClusterID, &h.ClusterName,
			&h.EstimatedTimeS, &h.ActualTimeS, &h.FilamentUsedM, &h.Status,
			&h.FailureReason, &h.StartedAt, &h.CompletedAt); err == nil {
			history = append(history, h)
		}
	}
	return history, nil
}

func (d *Database) GetPrinterWearStats() ([]PrinterWearStats, error) {
	rows, err := d.Conn.Query(`
		SELECT 
			p.id, p.name,
			COALESCE(ps.total_print_time_s, 0),
			COALESCE(ps.total_prints, 0),
			COALESCE(ps.successful_prints, 0),
			COALESCE(ps.failed_prints, 0),
			CASE 
				WHEN ps.total_prints > 0 THEN CAST(ps.successful_prints AS REAL) / ps.total_prints * 100
				ELSE 0
			END,
			CASE 
				WHEN ps.total_prints > 0 THEN CAST(ps.total_print_time_s AS REAL) / ps.total_prints
				ELSE 0
			END
		FROM printers p
		LEFT JOIN printer_stats ps ON p.id = ps.printer_id
		ORDER BY p.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := []PrinterWearStats{}
	for rows.Next() {
		var s PrinterWearStats
		if err := rows.Scan(&s.PrinterID, &s.PrinterName, &s.TotalPrintTimeS, &s.TotalPrints,
			&s.SuccessfulPrints, &s.FailedPrints, &s.SuccessRate, &s.AvgPrintTimeS); err == nil {
			// Calculate wear score: combination of total print time and failure rate
			// Lower score = less wear = better candidate for new jobs
			hoursPrinted := float64(s.TotalPrintTimeS) / 3600.0
			failurePenalty := float64(s.FailedPrints) * 100.0
			s.WearScore = hoursPrinted + failurePenalty
			stats = append(stats, s)
		}
	}
	return stats, nil
}

// --- Cluster and Printer Methods ---

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
		if err := rows.Scan(&c.ID, &c.Name, &c.TargetModel, &enabled); err != nil {
			continue
		}
		c.IsEnabled = enabled == 1
		c.Printers = []Printer{}
		clusters = append(clusters, c)
	}

	for i := range clusters {
		pRows, err := d.Conn.Query(`
			SELECT id, cluster_id, name, ip_address, status, printer_type, serial, access_code,
			       COALESCE(username, ''), COALESCE(password, ''), COALESCE(access_token, ''), 
			       COALESCE(secret, ''), COALESCE(extra, ''), COALESCE(created_at, '')
			FROM printers WHERE cluster_id = ?
		`, clusters[i].ID)
		if err != nil {
			log.Printf("Failed to query printers for cluster %d: %v", clusters[i].ID, err)
			continue
		}
		for pRows.Next() {
			var p Printer
			var createdAtStr string
			if err := pRows.Scan(&p.ID, &p.ClusterID, &p.Name, &p.IPAddress, &p.Status,
				&p.PrinterType, &p.Serial, &p.AccessCode, &p.Username, &p.Password,
				&p.AccessToken, &p.Secret, &p.Extra, &createdAtStr); err == nil {
				if createdAtStr != "" {
					if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
						p.CreatedAt = &t
					}
				}
				clusters[i].Printers = append(clusters[i].Printers, p)
			} else {
				log.Printf("Failed to scan printer row: %v", err)
			}
		}
		pRows.Close()
	}
	return clusters, nil
}

func (d *Database) AddPrinter(clusterID int64, name, ip, printerType, serial, accessCode, username, password, accessToken, secret, extra string) error {
	_, err := d.Conn.Exec(`
		INSERT INTO printers (cluster_id, name, ip_address, printer_type, serial, access_code, username, password, access_token, secret, extra)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, clusterID, name, ip, printerType, serial, accessCode, username, password, accessToken, secret, extra)
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
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM printers WHERE cluster_id = ?", id)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM clusters WHERE id = ?", id)
	if err != nil {
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

// --- Service Methods (Logs) ---

func (d *Database) SaveEvent(event api.NexusPacket) error {
	if event.Log != nil {
		_, err := d.Conn.Exec(
			"INSERT INTO system_logs (timestamp, module, level, message) VALUES (?, ?, ?, ?)",
			event.Log.Timestamp, event.Log.Module, event.Log.Level, event.Log.Message,
		)
		return err
	}
	return nil
}

// --- Analytics Methods ---

func (d *Database) GetSystemAnalytics() (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// Total prints
	var totalPrints int64
	d.Conn.QueryRow("SELECT COUNT(*) FROM print_history").Scan(&totalPrints)
	analytics["total_prints"] = totalPrints

	// Total print time
	var totalTimeS int64
	d.Conn.QueryRow("SELECT SUM(total_print_time_s) FROM printer_stats").Scan(&totalTimeS)
	analytics["total_print_time_hours"] = float64(totalTimeS) / 3600.0

	// Success rate
	var successful, failed int64
	d.Conn.QueryRow("SELECT SUM(successful_prints), SUM(failed_prints) FROM printer_stats").Scan(&successful, &failed)
	if total := successful + failed; total > 0 {
		analytics["success_rate"] = float64(successful) / float64(total) * 100
	} else {
		analytics["success_rate"] = 0.0
	}

	// Average print time
	var avgTime float64
	d.Conn.QueryRow("SELECT AVG(actual_time_s) FROM print_history WHERE status = 'COMPLETED'").Scan(&avgTime)
	analytics["avg_print_time_minutes"] = avgTime / 60.0

	return analytics, nil
}

// --- Filament Spool Methods ---

func (d *Database) AddSpool(material, color, colorHex string, totalWeight, remaining float32, assignedTo int64) error {
	_, err := d.Conn.Exec(`
		INSERT INTO spools (material, color, color_hex, total_weight_g, remaining_weight_g, assigned_printer_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, material, color, colorHex, totalWeight, remaining, assignedTo)
	return err
}

func (d *Database) GetSpools() ([]FilamentSpool, error) {
	rows, err := d.Conn.Query(`
		SELECT s.id, s.material, s.color, s.color_hex, s.total_weight_g, s.remaining_weight_g,
		       s.assigned_printer_id, p.name
		FROM spools s
		LEFT JOIN printers p ON s.assigned_printer_id = p.id
		ORDER BY s.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spools := []FilamentSpool{}
	for rows.Next() {
		var s FilamentSpool
		var printerName sql.NullString
		if err := rows.Scan(&s.ID, &s.Material, &s.Color, &s.ColorHex, &s.TotalWeight,
			&s.Remaining, &s.AssignedTo, &printerName); err == nil {
			if printerName.Valid {
				s.PrinterName = printerName.String
			}
			spools = append(spools, s)
		}
	}
	return spools, nil
}

func (d *Database) UpdateJobMetadata(jobID int64, thumbnailURL string, filamentWeight float32) error {
	_, err := d.Conn.Exec(`
		UPDATE print_jobs SET thumbnail_url = ?, filament_weight_g = ? WHERE id = ?
	`, thumbnailURL, filamentWeight, jobID)
	return err
}
