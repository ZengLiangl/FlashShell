package machine

import (
	"testing"

	"FlashDock/define"
)

func TestCommandNeedsSFTP(t *testing.T) {
	tests := []struct {
		name string
		cmd  define.Command
		want bool
	}{
		{
			name: "docker restart only",
			cmd: define.Command{
				Steps: define.StepList{{Command: "docker restart auth-service gateway"}},
			},
			want: false,
		},
		{
			name: "upload step",
			cmd: define.Command{
				Steps: define.StepList{{Command: "upload /tmp/app.jar /root/app/app.jar"}},
			},
			want: true,
		},
		{
			name: "mixed steps",
			cmd: define.Command{
				Steps: define.StepList{
					{Command: "upload /tmp/app.jar /root/app/app.jar"},
					{Command: "docker restart app"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CommandNeedsSFTP(tt.cmd); got != tt.want {
				t.Fatalf("CommandNeedsSFTP() = %v, want %v", got, tt.want)
			}
		})
	}
}
