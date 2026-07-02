package tokenmanager

import (
	"testing"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
)

func TestSecretData(t *testing.T) {
	const token = "ghs_installationtoken"

	tests := []struct {
		name      string
		basicAuth bool
		extraData map[string][]byte
		want      map[string]string
	}{
		{
			name: "token only",
			want: map[string]string{"token": token},
		},
		{
			name:      "basic auth only",
			basicAuth: true,
			want:      map[string]string{"username": BasicAuthUsername, "password": token},
		},
		{
			name:      "token with resolved extraData",
			extraData: map[string][]byte{"ca.crt": []byte("PEM")},
			want:      map[string]string{"token": token, "ca.crt": "PEM"},
		},
		{
			name:      "basic auth with resolved extraData",
			basicAuth: true,
			extraData: map[string][]byte{"ca.crt": []byte("PEM")},
			want:      map[string]string{"username": BasicAuthUsername, "password": token, "ca.crt": "PEM"},
		},
		{
			name:      "managed token key wins over resolved extraData",
			extraData: map[string][]byte{"token": []byte("spoofed")},
			want:      map[string]string{"token": token},
		},
		{
			name:      "managed basic-auth keys win over resolved extraData",
			basicAuth: true,
			extraData: map[string][]byte{"username": []byte("spoofed"), "password": []byte("spoofed")},
			want:      map[string]string{"username": BasicAuthUsername, "password": token},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner := &githubv1.Token{
				Spec: githubv1.TokenSpec{
					Secret: githubv1.TokenSecretSpec{
						BasicAuth: tt.basicAuth,
					},
				},
			}
			s := &tokenSecret{owner: owner, extraData: tt.extraData}

			got := s.SecretData(token)

			if len(got) != len(tt.want) {
				t.Fatalf("got %d keys %v, want %d keys %v", len(got), keys(got), len(tt.want), keysStr(tt.want))
			}
			for k, want := range tt.want {
				if string(got[k]) != want {
					t.Errorf("key %q: got %q, want %q", k, string(got[k]), want)
				}
			}
		})
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func keysStr(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
