package sheet

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/seaptc/seaptc/conference"
	"golang.org/x/oauth2/google"
)

func getBody(ctx context.Context, url string) (io.ReadCloser, error) {
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		io.Copy(os.Stdout, resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("fetch sheet returned %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func GetClasses(ctx context.Context, url string) ([]*conference.Class, error) {
	r, err := getBody(ctx, url)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return parseClasses(r)
}
