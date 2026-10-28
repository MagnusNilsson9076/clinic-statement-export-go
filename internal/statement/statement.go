package statement

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Appointment struct {
	ID       string
	Start    string
	Patient  string
	Status   string
	Reminder bool
}

type Line struct {
	Description string
	AmountCents int
}

type Account struct {
	ID           string
	Period       string
	Appointments []Appointment
	Lines        []Line
}

type Result struct {
	Markdown       string
	NotifyPatients []string
}

// Build decides which operational reminders are safe and formats the export input.
func Build(account Account) (Result, error) {
	if account.ID == "" || account.Period == "" {
		return Result{}, fmt.Errorf("account id and period are required")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Account statement\\n\\nAccount: %s\\nPeriod: %s\\n\\n", account.ID, account.Period)
	total := 0
	for _, line := range account.Lines {
		fmt.Fprintf(&b, "- %s: $%.2f\\n", line.Description, float64(line.AmountCents)/100)
		total += line.AmountCents
	}
	fmt.Fprintf(&b, "\\nTotal: $%.2f\\n", float64(total)/100)
	var notify []string
	for _, appointment := range account.Appointments {
		// Only confirmed visits with an explicit reminder opt-in are sent.
		if appointment.Status == "confirmed" && appointment.Reminder && appointment.Patient != "" {
			notify = append(notify, appointment.Patient)
		}
	}
	return Result{Markdown: b.String(), NotifyPatients: notify}, nil
}

func VerifyWebhook(secret, signature string, body []byte) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.TrimPrefix(signature, "sha256=")))
}

func UsageQuery(accountID, period string) string {
	values := url.Values{}
	values.Set("account_id", accountID)
	values.Set("period", period)
	return values.Encode()
}

func Dollars(cents int) string { return strconv.FormatFloat(float64(cents)/100, 'f', 2, 64) }
