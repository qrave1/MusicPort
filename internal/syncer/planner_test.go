package syncer

import (
	"context"
	"errors"
	"github.com/gojuno/minimock/v3"
	"testing"
)

func TestBuild(t *testing.T) {
	for _, tc := range []struct {
		name    string
		present bool
		failure error
	}{
		{name: "new"}, {name: "present", present: true}, {name: "index error", failure: errors.New("index")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			s := Source{Provider: "test", ID: "one", Format: "mp3"}
			index := NewIndexMock(mc).PresentMock.Expect(context.Background(), s).Return(tc.present, tc.failure)
			p, err := Build(context.Background(), index, []Track{{Source: &s}, {Source: &s}, {}})
			if !errors.Is(err, tc.failure) {
				t.Fatalf("error=%v", err)
			}
			if err != nil {
				return
			}
			if p.Unresolved != 1 {
				t.Fatal(p)
			}
			if tc.present {
				if p.Present != 1 || len(p.New) != 0 {
					t.Fatal(p)
				}
			} else if len(p.New) != 1 {
				t.Fatal(p)
			}
		})
	}
}

func TestBuildCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Build(ctx, NewIndexMock(minimock.NewController(t)), []Track{{}})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
