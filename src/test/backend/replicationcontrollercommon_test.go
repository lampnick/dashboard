// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"reflect"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned/testclient"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/sets"
)

func TestGetReplicationControllerPodInfo(t *testing.T) {
	cases := []struct {
		controller *api.ReplicationController
		pods       []api.Pod
		expected   ReplicationControllerPodInfo
	}{
		{
			&api.ReplicationController{
				Status: api.ReplicationControllerStatus{
					Replicas: 5,
				},
				Spec: api.ReplicationControllerSpec{
					Replicas: 4,
				},
			},
			[]api.Pod{
				{
					Status: api.PodStatus{
						Phase: api.PodRunning,
					},
				},
			},
			ReplicationControllerPodInfo{
				Current: 5,
				Desired: 4,
				Running: 1,
				Pending: 0,
				Failed:  0,
			},
		},
	}

	for _, c := range cases {
		actual := getReplicationControllerPodInfo(c.controller, c.pods)
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("getReplicaSetPodInfo(%#v, %#v) == \n%#v\nexpected \n%#v\n",
				c.controller, c.pods, actual, c.expected)
		}
	}
}

func TestToLabelSelector(t *testing.T) {
	requirement, _ := labels.NewRequirement("app", labels.InOperator, sets.NewString("test"))

	cases := []struct {
		selector map[string]string
		expected labels.LabelSelector
	}{
		{
			map[string]string{},
			labels.LabelSelector{},
		},
		{
			map[string]string{"app": "test"},
			labels.LabelSelector{*requirement},
		},
	}

	for _, c := range cases {
		actual, _ := toLabelSelector(c.selector)
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("toLabelSelector(%#v) == \n%#v\nexpected \n%#v\n",
				c.selector, actual, c.expected)
		}
	}
}

func TestGetServicesForDeletion(t *testing.T) {
	requirement, _ := labels.NewRequirement("app", labels.InOperator, sets.NewString("test"))

	cases := []struct {
		labelSelector             labels.Selector
		replicationControllerList *api.ReplicationControllerList
		expected                  *api.ServiceList
		expectedActions           []string
	}{
		{
			labels.LabelSelector{*requirement},
			&api.ReplicationControllerList{
				Items: []api.ReplicationController{
					{Spec: api.ReplicationControllerSpec{Selector: map[string]string{"app": "test"}}},
				},
			},
			&api.ServiceList{
				Items: []api.Service{
					{Spec: api.ServiceSpec{Selector: map[string]string{"app": "test"}}},
				},
			},
			[]string{"list", "list"},
		},
		{
			labels.LabelSelector{*requirement},
			&api.ReplicationControllerList{
				Items: []api.ReplicationController{
					{Spec: api.ReplicationControllerSpec{Selector: map[string]string{"app": "test"}}},
					{Spec: api.ReplicationControllerSpec{Selector: map[string]string{"app": "test"}}},
				},
			},
			&api.ServiceList{
				Items: []api.Service{
					{Spec: api.ServiceSpec{Selector: map[string]string{"app": "test"}}},
				},
			},
			[]string{"list"},
		},
		{
			labels.LabelSelector{*requirement},
			&api.ReplicationControllerList{},
			&api.ServiceList{
				Items: []api.Service{
					{Spec: api.ServiceSpec{Selector: map[string]string{"app": "test"}}},
				},
			},
			[]string{"list"},
		},
	}

	for _, c := range cases {
		fakeClient := testclient.NewSimpleFake(c.replicationControllerList, c.expected)

		getServicesForDeletion(fakeClient, c.labelSelector, "mock")

		actions := fakeClient.Actions()
		if len(actions) != len(c.expectedActions) {
			t.Errorf("Unexpected actions: %v, expected %d actions got %d", actions,
				len(c.expectedActions), len(actions))
			continue
		}

		for i, verb := range c.expectedActions {
			if actions[i].GetVerb() != verb {
				t.Errorf("Unexpected action: %+v, expected %s",
					actions[i], verb)
			}
		}
	}
}