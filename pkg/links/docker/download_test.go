package docker

import (
	"testing"
)

func TestDockerDownloadLink_parseImageName(t *testing.T) {
	dd := &DockerDownloadLink{}

	tests := []struct {
		name          string
		input         string
		expectedImage string
		expectedTag   string
	}{
		// Official Docker images (single name)
		{
			name:          "official image no tag",
			input:         "nginx",
			expectedImage: "library/nginx",
			expectedTag:   "latest",
		},
		{
			name:          "official image with tag",
			input:         "nginx:1.20",
			expectedImage: "library/nginx",
			expectedTag:   "1.20",
		},

		// DockerHub user/repo format (2-part, no registry)
		{
			name:          "dockerhub user repo no tag",
			input:         "grafana/grafana",
			expectedImage: "grafana/grafana",
			expectedTag:   "latest",
		},
		{
			name:          "dockerhub user repo with tag",
			input:         "praetorian/nebula:v1.0",
			expectedImage: "praetorian/nebula",
			expectedTag:   "v1.0",
		},

		// Custom registry with domain (2-part)
		{
			name:          "custom registry 2-part no tag",
			input:         "registry.com/myimage",
			expectedImage: "myimage",
			expectedTag:   "latest",
		},
		{
			name:          "custom registry 2-part with tag",
			input:         "ghcr.io/oj/gobuster:latest",
			expectedImage: "oj/gobuster",
			expectedTag:   "latest",
		},
		{
			name:          "registry with port 2-part",
			input:         "localhost:5000/myimage",
			expectedImage: "myimage",
			expectedTag:   "latest",
		},

		// Custom registry with org (3-part)
		{
			name:          "custom registry 3-part no tag",
			input:         "registry.com/org/repo",
			expectedImage: "org/repo",
			expectedTag:   "latest",
		},
		{
			name:          "custom registry 3-part with tag",
			input:         "gcr.io/my-project/my-image:v2.0",
			expectedImage: "my-project/my-image",
			expectedTag:   "v2.0",
		},
		{
			name:          "registry with port 3-part",
			input:         "localhost:5000/org/image:dev",
			expectedImage: "org/image",
			expectedTag:   "dev",
		},

		// ECR formats
		{
			name:          "ecr private registry",
			input:         "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-repo",
			expectedImage: "my-repo",
			expectedTag:   "latest",
		},
		{
			name:          "ecr private registry with tag",
			input:         "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-repo:v1.0",
			expectedImage: "my-repo",
			expectedTag:   "v1.0",
		},
		{
			name:          "ecr public registry",
			input:         "public.ecr.aws/my-namespace/my-repo",
			expectedImage: "my-repo",
			expectedTag:   "latest",
		},
		{
			name:          "ecr public registry with tag",
			input:         "public.ecr.aws/my-namespace/my-repo:latest",
			expectedImage: "my-repo",
			expectedTag:   "latest",
		},

		// Edge cases
		{
			name:          "complex tag with dash",
			input:         "user/repo:v1.0-alpine",
			expectedImage: "user/repo",
			expectedTag:   "v1.0-alpine",
		},
		{
			name:          "complex registry domain",
			input:         "my-registry.internal.company.com/team/project:sha-abc123",
			expectedImage: "team/project",
			expectedTag:   "sha-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, tag := dd.parseImageName(tt.input)

			if image != tt.expectedImage {
				t.Errorf("parseImageName(%q) image = %q, want %q", tt.input, image, tt.expectedImage)
			}

			if tag != tt.expectedTag {
				t.Errorf("parseImageName(%q) tag = %q, want %q", tt.input, tag, tt.expectedTag)
			}
		})
	}
}

// Test specific edge cases that were problematic
func TestDockerDownloadLink_parseImageName_EdgeCases(t *testing.T) {
	dd := &DockerDownloadLink{}

	t.Run("grafana/grafana should remain unchanged", func(t *testing.T) {
		image, tag := dd.parseImageName("grafana/grafana")
		expectedImage := "grafana/grafana"
		expectedTag := "latest"

		if image != expectedImage {
			t.Errorf("grafana/grafana should parse to %q, got %q", expectedImage, image)
		}

		if tag != expectedTag {
			t.Errorf("grafana/grafana tag should be %q, got %q", expectedTag, tag)
		}
	})

	t.Run("registry.com/image should extract image name", func(t *testing.T) {
		image, tag := dd.parseImageName("registry.com/myapp")
		expectedImage := "myapp"
		expectedTag := "latest"

		if image != expectedImage {
			t.Errorf("registry.com/myapp should parse to %q, got %q", expectedImage, image)
		}

		if tag != expectedTag {
			t.Errorf("registry.com/myapp tag should be %q, got %q", expectedTag, tag)
		}
	})

	t.Run("ECR registry should extract repo name", func(t *testing.T) {
		image, tag := dd.parseImageName("123456789012.dkr.ecr.us-east-2.amazonaws.com/nebula-test-secrets-repo-nyx0")
		expectedImage := "nebula-test-secrets-repo-nyx0"
		expectedTag := "latest"

		if image != expectedImage {
			t.Errorf("ECR image should parse to %q, got %q", expectedImage, image)
		}

		if tag != expectedTag {
			t.Errorf("ECR image tag should be %q, got %q", expectedTag, tag)
		}
	})
}
