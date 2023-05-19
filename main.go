package main

import (
	"MutatingWebhook/client"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

var Patches []PatchOperation
var (
	scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(scheme)
)

func main() {
	//Initializing the glog configuration
	flag.Parse()
	flag.Set("logtostderr", "false")
	flag.Set("alsologtostderr", "false")
	flag.Set("log_dir", "/var/log/myapp")
	go func() {
		for {
			glog.Flush()
			time.Sleep(1 * time.Second) // Log refreshed every 1 second
		}
	}()

	// Read certificate file and private key file
	cert, err := tls.LoadX509KeyPair("/etc/webhook/certs/tls.crt", "/etc/webhook/certs/tls.key")
	if err != nil {
		glog.Errorf("get cert fail.err is :", err)
		panic(err)
	}

	// Creating a TLS Configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		//ClientCAs:    caCertPool,
		//ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	// Create an HTTP server
	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
	}

	// start services
	http.Handle("/webhook", New(&applyNode{}))
	client.NewClientK8s()
	if err := server.ListenAndServeTLS("", ""); err != nil {
		glog.Errorf("server start fail,err is:", err)
		panic(err)
	}
}

type applyNode struct {
}

func (ch *applyNode) handler(w http.ResponseWriter, r *http.Request) {
	var writeErr error
	if bytes, err := webHookVerify(w, r); err != nil {
		glog.Errorf("Error handling webhook request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, writeErr = w.Write([]byte(err.Error()))
	} else {
		log.Print("Webhook request handled successfully")
		_, writeErr = w.Write(bytes)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
	if writeErr != nil {
		glog.Errorf("Could not write response: %v", writeErr)
	}
	return
}

func webHookVerify(w http.ResponseWriter, r *http.Request) (bytes []byte, err error) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, fmt.Errorf("invalid method %s, only POST requests are allowed", r.Method)
	}

	if contentType := r.Header.Get("Content-Type"); contentType != `application/json` {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("unsupported content type %s, only %s is supported", contentType, `application/json`)
	}

	var admissionReviewReq v1beta1.AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("r.Body parsing failed: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("request is nil")
	}
	glog.Infof("The structure information received by http is :", admissionReviewReq)
	//jsonData, err := json.Marshal(admissionReviewReq)
	//fmt.Println(string(jsonData))

	//You can add multiple services here, if you are modifying a node, go to the server of the node, if it is a pod you can go to the server of the pod
	node := corev1.Node{}
	obj, _, err := Codecs.UniversalDecoder().Decode(admissionReviewReq.Request.Object.Raw, nil, &node)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Request type is not Node.err is: %v", err)
	}

	if admissionReviewReq.Request.Namespace == metav1.NamespacePublic || admissionReviewReq.Request.Namespace == metav1.NamespaceSystem {
		glog.Infof("ns is a public resource and is prohibited from being modified.ns is :", admissionReviewReq.Request.Namespace)
		return nil, nil
	}
	//nodeInfo, _ := obj.(*corev1.Node)
	//jsonData, err := json.Marshal(nodeInfo)
	//fmt.Println(string(jsonData))
	if _, ok := obj.(*corev1.Node); ok {
		bytes, err = nodePatch(admissionReviewReq, node)
	}

	if err != nil {
		glog.Errorf("node server err,err is:", err)
	}
	return bytes, err
}

func nodePatch(admissionReviewReq v1beta1.AdmissionReview, nodeInfo corev1.Node) (bytes []byte, err error) {
	var finalCpu string
	var finalMem string

	// Query the CPU usage and memory usage of a node
	nodeMetrics, err := client.K8sClient.MetricsApi.MetricsV1beta1().NodeMetricses().Get(context.Background(), nodeInfo.Name, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	capacityCpu := nodeInfo.Status.Capacity.Cpu().MilliValue()
	capacityMem := nodeInfo.Status.Capacity.Memory().Value() / (1024 * 1024)
	//fmt.Println("capacityCpu cpu is", capacityCpu)
	//fmt.Println("capacityMem mem is", capacityMem)
	//fmt.Println("Allocatable cpu is", nodeInfo.Status.Allocatable.Cpu().MilliValue())
	//fmt.Println("Allocatable string cpu is", nodeInfo.Status.Allocatable.Cpu().String())
	//fmt.Println("Allocatable string mem is", nodeInfo.Status.Allocatable.Memory().String())
	//fmt.Println("Allocatable mem is", nodeInfo.Status.Allocatable.Memory().Value()/(1024*1024))
	//fmt.Println("nodeMetrics cpu is", nodeMetrics.Usage.Cpu().MilliValue())
	//fmt.Println("nodeMetrics mem is", nodeMetrics.Usage.Memory().Value()/(1024*1024))
	allocatableCpu := capacityCpu - nodeMetrics.Usage.Cpu().MilliValue()
	allocatableMem := capacityMem - nodeMetrics.Usage.Memory().Value()/(1024*1024)

	if allocatableCpu > nodeInfo.Status.Allocatable.Cpu().MilliValue() {
		floatCpu := math.Round(float64(allocatableCpu / 1000))
		finalCpu = strconv.FormatFloat(floatCpu, 'f', -1, 64)
	} else {
		finalCpu = nodeInfo.Status.Allocatable.Cpu().String()
	}
	if allocatableMem > nodeInfo.Status.Allocatable.Memory().Value()/(1024*1024) {
		finalMem = strconv.Itoa(int(allocatableMem*1024)) + "Ki"
	} else {
		finalMem = nodeInfo.Status.Allocatable.Memory().String()
	}
	admissionReviewResponse := v1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &v1beta1.AdmissionResponse{
			UID: admissionReviewReq.Request.UID,
		},
	}
	//Oversold logic can be added here
	patchOps := append(Patches, getPatchItem("replace", "/status/allocatable/cpu", string(finalCpu)), getPatchItem("replace", "/status/allocatable/memory", finalMem))
	patchBytes, err := json.Marshal(patchOps)
	admissionReviewResponse.Response.Allowed = true
	admissionReviewResponse.Response.Patch = patchBytes
	admissionReviewResponse.Response.PatchType = func() *v1beta1.PatchType {
		pt := v1beta1.PatchTypeJSONPatch
		return &pt
	}()

	// Return the AdmissionReview with a response as JSON.
	bytes, err = json.Marshal(&admissionReviewResponse)
	return
}

func getPatchItem(op string, path string, val interface{}) PatchOperation {
	return PatchOperation{
		Op:    op,
		Path:  path,
		Value: val,
	}
}

type Handler interface {
	handler(w http.ResponseWriter, r *http.Request)
}

type HandleProxy struct {
	handler Handler
}

func New(handler Handler) *HandleProxy {
	return &HandleProxy{
		handler: handler,
	}
}

//The Handle needs to implement ServeHTTP
func (h *HandleProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	h.handler.handler(w, r)
}

