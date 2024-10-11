package worker

import (
	"math/big"
	"reflect"
	"testing"
)

func TestStringToBigInt(t *testing.T) {
	type args struct {
		numberString string
		bump         bool
	}
	tests := []struct {
		name    string
		args    args
		want    *big.Int
		wantErr bool
	}{
		{
			name: "no bump",
			args: args{
				numberString: "100",
				bump:         false,
			},
			want:    big.NewInt(100),
			wantErr: false,
		},
		{
			name: "bump",
			args: args{
				numberString: "100",
				bump:         true,
			},
			want:    big.NewInt(105),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringToBigInt(tt.args.numberString, tt.args.bump)
			if (err != nil) != tt.wantErr {
				t.Errorf("StringToBigInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringToBigInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
