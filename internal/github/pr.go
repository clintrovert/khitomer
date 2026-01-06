package github

import (
	"fmt"
	"strings"
)

// GenerateBranchName generates a branch name from Jira ticket ID and title
func GenerateBranchName(ticketID, title string) string {
	shortTitle := truncateString(title, 30)
	return "khitomer/" + ticketID + "-" + sanitizeBranchName(shortTitle)
}

// GeneratePRTitle generates a PR title from Jira ticket ID and title
func GeneratePRTitle(ticketID, title string) string {
	return ticketID + ": " + title
}

// GeneratePRDescription generates a PR description from task and plan information
func GeneratePRDescription(jiraTicketID, description, planSummary string, steps []string) string {
	var sb strings.Builder
	
	sb.WriteString("## Implementation for " + jiraTicketID + "\n\n")
	sb.WriteString("**Jira Ticket:** " + jiraTicketID + "\n")
	sb.WriteString("**Description:** " + description + "\n\n")
	sb.WriteString("## Implementation Plan\n\n")
	sb.WriteString(planSummary + "\n\n")
	
	if len(steps) > 0 {
		sb.WriteString("## Steps\n\n")
		for i, step := range steps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}
	
	return sb.String()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func sanitizeBranchName(s string) string {
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else if r == ' ' {
			result.WriteRune('-')
		}
	}
	return result.String()
}

