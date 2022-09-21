/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestLegacyIsMasterNode(t *testing.T) {
	testcases := map[string]struct {
		Name   string
		Labels map[string]string
		expect bool
	}{
		"node with node label key": {
			Labels: map[string]string{keyMasterNodeLabel: ""},
			expect: true,
		},
		"node with node label key value": {
			Labels: map[string]string{keyMasterNodeLabel: "true"},
			expect: true,
		},
		"node without node label": {
			Labels: map[string]string{},
			expect: false,
		},
	}

	for _, tc := range testcases {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Labels: tc.Labels},
		}
		result := LegacyIsMasterNode(node)
		assert.Equal(t, tc.expect, result)
	}
}
