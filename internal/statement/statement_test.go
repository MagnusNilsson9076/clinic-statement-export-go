package statement

import "testing"

func TestBuildOnlyNotifiesOptedInConfirmedPatients(t *testing.T) {
	cases := []struct {
		name         string
		appointments []Appointment
		want         []string
	}{
		{name: "only confirmed opt-in", appointments: []Appointment{
			{ID: "a1", Patient: "p-1", Status: "confirmed", Reminder: true},
			{ID: "a2", Patient: "p-2", Status: "confirmed", Reminder: false},
			{ID: "a3", Patient: "p-3", Status: "cancelled", Reminder: true},
		}, want: []string{"p-1"}},
		{name: "no reminder", appointments: []Appointment{{ID: "a4", Patient: "p-4", Status: "confirmed"}}, want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Build(Account{ID: "acct-42", Period: "2026-10", Lines: []Line{{Description: "consultation", AmountCents: 12500}}, Appointments: tc.appointments})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.NotifyPatients) != len(tc.want) {
				t.Fatalf("notifications = %v, want %v", result.NotifyPatients, tc.want)
			}
			for i := range tc.want {
				if result.NotifyPatients[i] != tc.want[i] {
					t.Fatalf("patient = %q, want %q", result.NotifyPatients[i], tc.want[i])
				}
			}
		})
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	body := []byte(`{"event":"appointment.confirmed"}`)
	if !VerifyWebhook("local-secret", "sha256=41e5ecce8a65ca62b5ed8507282c4df0d5172f171522341e558c56f20c66d97f", body) {
		t.Fatal("expected the sample signature to verify")
	}
}
