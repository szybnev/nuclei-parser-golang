package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
)

type Issue struct {
	TemplateID string `json:"template-id"`
	Info       struct {
		Name     string `json:"name"`
		Severity string `json:"severity"`
	} `json:"info"`
	Host      string `json:"host"`
	MatchedAt string `json:"matched-at"`
}

var severityOrder = map[string]int{
	"critical": 1,
	"high":     2,
	"medium":   3,
	"low":      4,
	"info":     5,
	"unknown":  6,
}

func severityKey(severity string) int {
	if order, exists := severityOrder[severity]; exists {
		return order
	}
	return severityOrder["unknown"]
}

func generateHTML(issues []Issue, filename string) {
	const tpl = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nuclei Scan Result</title>
    <style>
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            border: 1px solid black;
            padding: 8px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
    </style>
</head>
<body>
    <h1>Scan Result</h1>
    <table>
        <tr>
            <th>Template ID</th>
            <th>Name</th>
            <th>Severity</th>
            <th>Host</th>
            <th>Matched At</th>
        </tr>
        {{range .}}
        <tr>
            <td>{{.TemplateID}}</td>
            <td>{{.Info.Name}}</td>
            <td>{{.Info.Severity}}</td>
            <td>{{.Host}}</td>
            <td>{{.MatchedAt}}</td>
        </tr>
        {{end}}
    </table>
</body>
</html>`

	t, err := template.New("report").Parse(tpl)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create HTML file: %v", err)
	}
	defer file.Close()

	err = t.Execute(file, issues)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}

func printTable(issues []Issue) {
	fmt.Printf("+--------------------------+--------------------------------------------------+----------+--------------------+----------------------------------------------------------------+\n")
	fmt.Printf("| Template ID              | Name                                             | Severity | Host               | Matched At                                                      |\n")
	fmt.Printf("+--------------------------+--------------------------------------------------+----------+--------------------+----------------------------------------------------------------+\n")
	for _, issue := range issues {
		fmt.Printf("| %-24s | %-48s | %-8s | %-18s | %-62s |\n",
			issue.TemplateID, issue.Info.Name, issue.Info.Severity, issue.Host, issue.MatchedAt)
		fmt.Printf("+--------------------------+--------------------------------------------------+----------+--------------------+----------------------------------------------------------------+\n")
	}
}

func main() {
	htmlFlag := flag.Bool("html", false, "Generate HTML output instead of terminal output")
	dataFile := flag.String("d", "", "Path to the JSON data file")
	severityFlag := flag.String("s", "", "Filter results by severity (comma separated)")
	flag.Parse()

	if *dataFile == "" {
		log.Fatal("Path to the JSON data file is required")
	}

	severities := []string{}
	if *severityFlag != "" {
		severities = strings.Split(*severityFlag, ",")
		for i, s := range severities {
			severities[i] = strings.TrimSpace(s)
		}
	}

	data, err := ioutil.ReadFile(*dataFile)
	if err != nil {
		log.Fatalf("Failed to read data file: %v", err)
	}

	var issues []Issue
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var issue Issue
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			log.Printf("Failed to parse line: %s\n%v", line, err)
			continue
		}
		issues = append(issues, issue)
	}

	log.Printf("Loaded %d issues from the data file.\n", len(issues))

	if len(severities) > 0 {
		var filteredIssues []Issue
		for _, issue := range issues {
			for _, severity := range severities {
				if strings.EqualFold(issue.Info.Severity, severity) {
					filteredIssues = append(filteredIssues, issue)
					break
				}
			}
		}
		issues = filteredIssues
		log.Printf("Filtered issues: %d with severities %s\n", len(issues), strings.Join(severities, ", "))
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if severityKey(issues[i].Info.Severity) == severityKey(issues[j].Info.Severity) {
			return strings.Compare(issues[i].TemplateID, issues[j].TemplateID) < 0
		}
		return severityKey(issues[i].Info.Severity) < severityKey(issues[j].Info.Severity)
	})

	if *htmlFlag {
		generateHTML(issues, "output.html")
		log.Println("HTML generated")
	} else {
		printTable(issues)
	}
}
