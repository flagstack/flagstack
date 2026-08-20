package sdkconfig

import "testing"

func TestInvalidationHubScopesConfigurationAndCredentialEvents(t *testing.T) {
	hub := NewInvalidationHub()
	production := Credential{ID: "credential-production", ProjectID: "project-1", EnvironmentID: "environment-production"}
	staging := Credential{ID: "credential-staging", ProjectID: "project-1", EnvironmentID: "environment-staging"}
	otherProject := Credential{ID: "credential-other", ProjectID: "project-2", EnvironmentID: "environment-production"}

	productionEvents, cancelProduction := hub.Subscribe(production)
	defer cancelProduction()
	stagingEvents, cancelStaging := hub.Subscribe(staging)
	defer cancelStaging()
	otherEvents, cancelOther := hub.Subscribe(otherProject)
	defer cancelOther()

	hub.Publish(Invalidation{ProjectID: "project-1", EnvironmentID: "environment-production"})
	assertInvalidationReceived(t, productionEvents)
	assertNoInvalidation(t, stagingEvents)
	assertNoInvalidation(t, otherEvents)

	hub.Publish(Invalidation{ProjectID: "project-1"})
	assertInvalidationReceived(t, productionEvents)
	assertInvalidationReceived(t, stagingEvents)
	assertNoInvalidation(t, otherEvents)

	hub.Publish(Invalidation{ProjectID: "project-1", EnvironmentID: "environment-staging", CredentialID: "credential-staging"})
	assertNoInvalidation(t, productionEvents)
	assertInvalidationReceived(t, stagingEvents)
	assertNoInvalidation(t, otherEvents)
}

func TestInvalidationHubCoalescesPendingSignals(t *testing.T) {
	hub := NewInvalidationHub()
	events, cancel := hub.Subscribe(Credential{ID: "credential-1", ProjectID: "project-1", EnvironmentID: "environment-1"})
	defer cancel()

	for range 10 {
		hub.Publish(Invalidation{ProjectID: "project-1", EnvironmentID: "environment-1"})
	}
	assertInvalidationReceived(t, events)
	assertNoInvalidation(t, events)
}

func assertInvalidationReceived(t *testing.T, events <-chan Invalidation) {
	t.Helper()
	select {
	case <-events:
	default:
		t.Fatal("expected invalidation signal")
	}
}

func assertNoInvalidation(t *testing.T, events <-chan Invalidation) {
	t.Helper()
	select {
	case event := <-events:
		t.Fatalf("unexpected invalidation %#v", event)
	default:
	}
}
