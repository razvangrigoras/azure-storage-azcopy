// Copyright © Microsoft <wastore@microsoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"context"
	"net/url"
	"path/filepath"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-file-go/azfile"
)

// Enumerates an entire files account, looking into each matching share as it goes
type fileAccountTraverser struct {
	accountURL   azfile.ServiceURL
	p            pipeline.Pipeline
	ctx          context.Context
	sharePattern string

	// a generic function to notify that a new stored object has been enumerated
	incrementEnumerationCounter func()
}

func (t *fileAccountTraverser) traverse(processor objectProcessor, filters []objectFilter) error {
	marker := azfile.Marker{}
	for marker.NotDone() {
		resp, err := t.accountURL.ListSharesSegment(t.ctx, marker, azfile.ListSharesOptions{})

		if err != nil {
			return err
		}

		for _, v := range resp.ShareItems {
			// Match a pattern for the share name and the share name only
			if t.sharePattern != "" {
				if ok, err := filepath.Match(t.sharePattern, v.Name); err != nil {
					// Break if the pattern is invalid
					return err
				} else if !ok {
					// Ignore the share if it doesn't match the pattern.
					continue
				}
			}

			shareURL := t.accountURL.NewShareURL(v.Name).URL()
			shareTraverser := newFileTraverser(&shareURL, t.p, t.ctx, true, t.incrementEnumerationCounter)

			middlemanProcessor := initContainerDecorator(v.Name, processor)

			err = shareTraverser.traverse(middlemanProcessor, filters)

			if err != nil {
				return err
			}
		}

		marker = resp.NextMarker
	}

	return nil
}

func newFileAccountTraverser(rawURL *url.URL, p pipeline.Pipeline, ctx context.Context, incrementEnumerationCounter func()) (t *fileAccountTraverser) {
	fURLparts := azfile.NewFileURLParts(*rawURL)
	sPattern := fURLparts.ShareName

	if fURLparts.ShareName != "" {
		fURLparts.ShareName = ""
	}

	t = &fileAccountTraverser{p: p, ctx: ctx, incrementEnumerationCounter: incrementEnumerationCounter, accountURL: azfile.NewServiceURL(fURLparts.URL(), p), sharePattern: sPattern}
	return
}