package build

import (
	"fmt"
	"testing"
)

func TestCleanQuerySql(t *testing.T) {
	type args struct {
		raw string
	}
	tests := []struct {
		name        string
		args        args
		wantCleaned string
	}{
		{
			name:        "test1",
			args:        args{raw: "\\u0000\\u0001select 1"},
			wantCleaned: "select 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println([]byte("select *"))
			if gotCleaned := CleanQuerySql(tt.args.raw); gotCleaned != tt.wantCleaned {
				t.Errorf("CleanQuerySql() = %v, want %v", gotCleaned, tt.wantCleaned)
			}
		})
	}
}
