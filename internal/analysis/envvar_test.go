package analysis

import (
    "testing"
)

func TestAnalyze_EnvVarInReverseProxyArg(t *testing.T) {
    // The exact pattern from the user's report
    src := "example.com {\n\treverse_proxy /admin-api/* https://{$LOCALHOST_GATEWAY}:3355 {\n\t\theader_up X-Real-IP {http.request.remote.host}\n\t}\n}\n"
    diags := analyze(src)
    if len(diags) != 0 {
        t.Errorf("env var in reverse_proxy URL: expected no diagnostics, got %d: %v", len(diags), diags)
    }
}

func TestAnalyze_EnvVarStandaloneArg(t *testing.T) {
    src := "example.com {\n\treverse_proxy {$UPSTREAM}\n}\n"
    diags := analyze(src)
    if len(diags) != 0 {
        t.Errorf("standalone env var arg: expected no diagnostics, got %d: %v", len(diags), diags)
    }
}

func TestAnalyze_EnvVarInSiteAddress(t *testing.T) {
    src := "https://{$DOMAIN}:8080 {\n\treverse_proxy /api/* {$BACKEND}\n}\n"
    diags := analyze(src)
    if len(diags) != 0 {
        t.Errorf("env var in site address: expected no diagnostics, got %d: %v", len(diags), diags)
    }
}

func TestAnalyze_EnvVarStandaloneAddress(t *testing.T) {
    src := "{$SITE_ADDR} {\n\treverse_proxy /api/* {$BACKEND}\n}\n"
    diags := analyze(src)
    if len(diags) != 0 {
        t.Errorf("env var as site address: expected no diagnostics, got %d: %v", len(diags), diags)
    }
}
