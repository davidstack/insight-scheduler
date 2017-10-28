// Copyright 2016 Google Inc. All Rights Reserved.
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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	//apiHost           = "10.110.17.45:8080"
	bindingsEndpoint  = "/api/v1/namespaces/%s/pods/%s/binding/"
	eventsEndpoint    = "/api/v1/namespaces/%s/events"
	nodesEndpoint     = "/api/v1/nodes"
	podsEndpoint      = "/api/v1/pods"
	watchPodsEndpoint = "/api/v1/watch/pods"
)
func postEvent(event Event,namespace string) error {

	var b []byte
	body := bytes.NewBuffer(b)
	err := json.NewEncoder(body).Encode(event)
	if err != nil {
		return err
	}

	request := &http.Request{
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        make(http.Header),
		Method:        http.MethodPost,
		URL: &url.URL{
			Host:   apiHost,
			Path:   fmt.Sprintf(eventsEndpoint,namespace),
			Scheme: "http",
		},
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return errors.New("Event: Unexpected HTTP status code" + resp.Status)
	}
	return nil
}

func getNodes() (*NodeList, error) {
	var nodeList NodeList

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:   apiHost,
			Path:   nodesEndpoint,
			Scheme: "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&nodeList)
	if err != nil {
		return nil, err
	}

	return &nodeList, nil
}

func watchUnscheduledPods() (<-chan Pod, <-chan error) {
	pods := make(chan Pod)
	errc := make(chan error, 1)

	v := url.Values{}
	v.Set("labelSelector", "type=insight-statefulset")
	v.Set("fieldSelector", "spec.nodeName=")
	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     apiHost,
			Path:     watchPodsEndpoint,
			RawQuery: v.Encode(),
			Scheme:   "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	go func() {
		for {
			resp, err := http.DefaultClient.Do(request)
			if err != nil {
				errc <- err
				time.Sleep(5 * time.Second)
				continue
			}

			if resp.StatusCode != 200 {
				errc <- errors.New("Invalid status code: " + resp.Status)
				time.Sleep(5 * time.Second)
				continue
			}

			decoder := json.NewDecoder(resp.Body)
			for {
				var event PodWatchEvent
				err = decoder.Decode(&event)
				if err != nil {
					errc <- err
					break
				}

				if event.Type == "ADDED" {
					pods <- event.Object
				}
			}
		}
	}()

	return pods, errc
}

func getUnscheduledPods() ([]*Pod, error) {
	var podList PodList
	unscheduledPods := make([]*Pod, 0)

	v := url.Values{}
	v.Set("labelSelector", "type=insight-statefulset")
	v.Set("fieldSelector", "spec.nodeName=")
	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     apiHost,
			Path:     podsEndpoint,
			RawQuery: v.Encode(),
			Scheme:   "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return unscheduledPods, err
	}
	err = json.NewDecoder(resp.Body).Decode(&podList)
	if err != nil {
		return unscheduledPods, err
	}
    log.Println("watch pod is "+StructToJson(podList))
	for _, pod := range podList.Items {
		if pod.Spec.SchedulerName == schedulerName {
			unscheduledPods = append(unscheduledPods, &pod)
		}
	}

	return unscheduledPods, nil
}

//func getPods() (*PodList, error) {
//	var podList PodList
//
//	v := url.Values{}
//	v.Add("fieldSelector", "status.phase=Running")
//	v.Add("fieldSelector", "status.phase=Pending")
//
//	request := &http.Request{
//		Header: make(http.Header),
//		Method: http.MethodGet,
//		URL: &url.URL{
//			Host:     apiHost,
//			Path:     podsEndpoint,
//			RawQuery: v.Encode(),
//			Scheme:   "http",
//		},
//	}
//	request.Header.Set("Accept", "application/json, */*")
//
//	resp, err := http.DefaultClient.Do(request)
//	if err != nil {
//		return nil, err
//	}
//	err = json.NewDecoder(resp.Body).Decode(&podList)
//	if err != nil {
//		return nil, err
//	}
//	return &podList, nil
//}

type ResourceUsage struct {
	CPU int
}

func fit(pod *Pod) ([]Node, error) {
	nodeList, err := getNodes()
	if err != nil {
		return nil, err
	}
	//log.Println("nodeslist is {}",StructToJson(nodeList))
	//format container destname as C idefiner
    nodeDestName :=""
	containerEnv :=pod.Spec.Containers[0].Env
	containerHostName :=pod.Metadata.Annotations["pod.beta.kubernetes.io/hostname"]
	containerHostNameUpper :=strings.Replace(containerHostName,"-","_",-1)
	//log.Println("upper name is"+containerHostNameUpper)
    for _ ,env :=range containerEnv{
    	if env.Name==containerHostNameUpper{
			nodeDestName=env.Value
		}
	}
	log.Println("containerHostNameUpper name is "+containerHostNameUpper)
	log.Println("dest node name is "+nodeDestName)

	//get node info
	var nodes []Node
	fitFailures := make([]string, 0)
	for _, node := range nodeList.Items {
		//log.Println("node.Metadata.Name is "+node.Metadata.Name)
		if node.Metadata.Name==nodeDestName{
			//log.Println("node.Conditions is ",StructToJson(node.Status.Conditions))
			for _,nodeCondition :=range node.Status.Conditions{
				if nodeCondition.Type=="Ready"&& nodeCondition.Status=="True"{
					nodes = append(nodes, node)
					break
				}
			}
			break
		}

	}

	if len(nodes) == 0 {
		// Emit a Kubernetes event that the Pod was scheduled unsuccessfully.
		timestamp := time.Now().UTC().Format(time.RFC3339)
		event := Event{
			Count:          1,
			Message:        fmt.Sprintf("pod (%s) failed to fit in any node\n%s", pod.Metadata.Name, strings.Join(fitFailures, "\n")),
			Metadata:       Metadata{GenerateName: pod.Metadata.Name + "-"},
			Reason:         "FailedScheduling",
			LastTimestamp:  timestamp,
			FirstTimestamp: timestamp,
			Type:           "Warning",
			Source:         EventSource{Component: "insight-scheduler"},
			InvolvedObject: ObjectReference{
				Kind:      "Pod",
				Name:      pod.Metadata.Name,
				Namespace: pod.Metadata.Namespace,
				Uid:       pod.Metadata.Uid,
			},
		}

		postEvent(event,pod.Metadata.Namespace)
	}

	return nodes, nil
}

func bind(pod *Pod, node Node) error {
	binding := Binding{
		ApiVersion: "v1",
		Kind:       "Binding",
		Metadata:   Metadata{Name: pod.Metadata.Name},
		Target: Target{
			ApiVersion: "v1",
			Kind:       "Node",
			Name:       node.Metadata.Name,
		},
	}

	var b []byte
	body := bytes.NewBuffer(b)
	err := json.NewEncoder(body).Encode(binding)
	if err != nil {
		return err
	}

	request := &http.Request{
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        make(http.Header),
		Method:        http.MethodPost,
		URL: &url.URL{
			Host:   apiHost,
			Path:   fmt.Sprintf(bindingsEndpoint,pod.Metadata.Namespace,pod.Metadata.Name),
			Scheme: "http",
		},
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return errors.New("Binding: Unexpected HTTP status code" + resp.Status)
	}

	// Emit a Kubernetes event that the Pod was scheduled successfully.
	message := fmt.Sprintf("Successfully assigned %s to %s", pod.Metadata.Name, node.Metadata.Name)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	event := Event{
		Count:          1,
		Message:        message,
		Metadata:       Metadata{GenerateName: pod.Metadata.Name + "-"},
		Reason:         "Scheduled",
		LastTimestamp:  timestamp,
		FirstTimestamp: timestamp,
		Type:           "Normal",
		Source:         EventSource{Component: "insight-scheduler"},
		InvolvedObject: ObjectReference{
			Kind:      "Pod",
			Name:      pod.Metadata.Name,
			Namespace: pod.Metadata.Namespace,
			Uid:       pod.Metadata.Uid,
		},
	}
	log.Println(message)
	return postEvent(event,pod.Metadata.Namespace)
}
