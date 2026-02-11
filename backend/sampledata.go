package main

import (
	"fmt"
	"math/rand"
	"time"
)

// SampleRecord represents a single data record for the table
type SampleRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	Status      string `json:"status"`
	Category    string `json:"category"`
}

var categories = []string{
	"Engineering", "Marketing", "Sales", "Operations", "Support", "Finance", "HR", "Product",
}

var statuses = []string{
	"Active", "Pending", "Completed", "On Hold", "Cancelled",
}

var adjectives = []string{
	"Strategic", "Innovative", "Critical", "Quarterly", "Annual", "Monthly", "Priority",
	"Collaborative", "Automated", "Enhanced", "Optimized", "Integrated", "Advanced",
}

var nouns = []string{
	"Project", "Initiative", "Campaign", "Analysis", "Review", "Assessment", "Migration",
	"Implementation", "Deployment", "Integration", "Upgrade", "Rollout", "Launch",
}

var descriptions = []string{
	"Implementing new features and improvements",
	"Analyzing performance metrics and KPIs",
	"Coordinating cross-team collaboration",
	"Reviewing and updating documentation",
	"Planning and executing quarterly goals",
	"Optimizing resource allocation",
	"Conducting stakeholder meetings",
	"Developing strategic roadmap",
	"Evaluating vendor solutions",
	"Training team members on new tools",
	"Migrating legacy systems",
	"Enhancing security protocols",
	"Streamlining business processes",
	"Building customer relationships",
	"Improving operational efficiency",
}

// generateSampleData creates a slice of sample records
func generateSampleData() []SampleRecord {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	numRecords := 50 + r.Intn(51) // 50-100 records

	records := make([]SampleRecord, numRecords)
	now := time.Now()

	for i := 0; i < numRecords; i++ {
		// Generate a random date within the last 365 days
		daysAgo := r.Intn(365)
		createdAt := now.AddDate(0, 0, -daysAgo)

		// Generate a meaningful name
		adj := adjectives[r.Intn(len(adjectives))]
		noun := nouns[r.Intn(len(nouns))]
		name := fmt.Sprintf("%s %s %d", adj, noun, 1000+i)

		records[i] = SampleRecord{
			ID:          fmt.Sprintf("REC-%05d", 10000+i),
			Name:        name,
			Description: descriptions[r.Intn(len(descriptions))],
			CreatedAt:   createdAt.Format(time.RFC3339),
			Status:      statuses[r.Intn(len(statuses))],
			Category:    categories[r.Intn(len(categories))],
		}
	}

	return records
}
