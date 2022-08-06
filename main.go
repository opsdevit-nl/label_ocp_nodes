package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	// kubernetes "github.com/openshift/client-go"
	"k8s.io/client-go/tools/clientcmd"
)

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func main() {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	config, err := kubeconfig.ClientConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset := kubernetes.NewForConfigOrDie(config)
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	vms := getVMS()
	for _, n := range nodes.Items {
		// labels := n.ObjectMeta.Labels
		// labels["esx-node"] = vms[n.Name]
		// fmt.Printf("VM/OCP node: %v on ESX node: %v\n", n.Name, n.Labels["esx-node"])
		payload := []patchStringValue{{
			Op:    "replace",
			Path:  "/metadata/labels/esx-node",
			Value: vms[n.Name],
		}}
		payloadBytes, _ := json.Marshal(payload)

		// print het hele object in JSON (in dit geval de node)
		// nodeBytes, _ := json.Marshal(n)
		// var prettyJSON bytes.Buffer
		// err := json.Indent(&prettyJSON, nodeBytes, "", "  ")
		// if err != nil {
		// 	log.Println("violation", string(prettyJSON.Bytes()))
		// }

		// log.Println(string(prettyJSON.Bytes()))

		var updateErr error
		// TODO: timestamp in logregel, zorg dat gechecked wordt of vmware reageert, kubeconfig werkt voor openshift, stub vmware code weg is
		labels := n.ObjectMeta.Labels
		if labels["esx-node"] != vms[n.Name] && vms[n.Name] != "" {
			_, updateErr = clientset.CoreV1().Nodes().Patch(context.TODO(), n.GetName(), types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
			if updateErr == nil {
				log.Printf("Node %s labelled successfully with esx-node %s. \n", n.GetName(), vms[n.Name])
			} else {
				fmt.Println(updateErr)
			}
		}
	}
}
