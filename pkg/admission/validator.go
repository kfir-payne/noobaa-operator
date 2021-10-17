package admission

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/system"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//GrumpyServerHandler listen to admission requests and serve responses
type GrumpyServerHandler struct {
}

func (gs *GrumpyServerHandler) serve(w http.ResponseWriter, r *http.Request) {

	var body []byte
	log := util.Logger()
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		log.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	log.Info("Received request")

	if r.URL.Path != "/validate" {
		log.Error("no validate")
		http.Error(w, "no validate", http.StatusBadRequest)
		return
	}

	arRequest := admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, &arRequest); err != nil {
		log.Error("incorrect body")
		http.Error(w, "incorrect body", http.StatusBadRequest)
	}

	raw := arRequest.Request.OldObject.Raw
	deletedBs := nbv1.BackingStore{}
	if err := json.Unmarshal(raw, &deletedBs); err != nil {
		log.Error("error deserializing BS")
		return
	}

	sysClient, err := system.Connect(false)
	if err != nil {
		log.Error("faild to load noobaa system connection info")
		http.Error(w, "faild to load noobaa system connection info", http.StatusBadRequest)
	}

	systemInfo, err := sysClient.NBClient.ReadSystemAPI()
	if err != nil {
		log.Error("failed to call ReadSystemInfo API")
		http.Error(w, "failed to call ReadSystemInfo API", http.StatusBadRequest)
	}

	arResponse := admissionv1.AdmissionReview{}
	allowed := false
	message := ""

	for _, pool := range systemInfo.Pools {
		if pool.Name == deletedBs.Name {
			if pool.Undeletable == "IS_BACKINGSTORE" {
				allowed = true
				message = "Allowed"
			} else {
				allowed = false
				message = fmt.Sprintf("Cannot complete because pool %q in %q state", pool.Name, pool.Undeletable)
			}
		}
	}

	if message == "" {
		message = fmt.Sprintf("BackingStore %q not found", deletedBs.Name)
	}

	arResponse = admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind: "AdmissionReview", 
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionv1.AdmissionResponse{
			Allowed: allowed,
			UID: arRequest.Request.UID,
			Result: &metav1.Status{
				Message: message,
			},
		},
	}

	resp, err := json.Marshal(arResponse)
	if err != nil {
		log.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	log.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		log.Infof("Can't write response")
		log.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// func (gs *GrumpyServerHandler) serve(w http.ResponseWriter, r *http.Request) {
// 	var body []byte
// 	log := util.Logger()
// 	if r.Body != nil {
// 		if data, err := ioutil.ReadAll(r.Body); err == nil {
// 			body = data
// 		}
// 	}
// 	if len(body) == 0 {
// 		log.Error("empty body")
// 		http.Error(w, "empty body", http.StatusBadRequest)
// 		return
// 	}
// 	log.Info("Received request")

// 	if r.URL.Path != "/validate" {
// 		log.Error("no validate")
// 		http.Error(w, "no validate", http.StatusBadRequest)
// 		return
// 	}

// 	arRequest := admissionv1.AdmissionReview{}
// 	if err := json.Unmarshal(body, &arRequest); err != nil {
// 		log.Error("incorrect body")
// 		http.Error(w, "incorrect body", http.StatusBadRequest)
// 	}

// 	raw := arRequest.Request.Object.Raw
// 	pod := v1.Pod{}
// 	if err := json.Unmarshal(raw, &pod); err != nil {
// 		log.Error("error deserializing pod")
// 		return
// 	}
// 	if pod.Name != "smooth-app" {
// 		return
// 	}

// 	arResponse := admissionv1.AdmissionReview{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind: "AdmissionReview", 
// 			APIVersion: "admission.k8s.io/v1",
// 		},
// 		Response: &admissionv1.AdmissionResponse{
// 			Allowed: false,
// 			Result: &metav1.Status{
// 				Message: "Keep calm and not add more crap in the cluster!",
// 			},
// 			UID: arRequest.Request.UID,
// 		},
// 	}
// 	resp, err := json.Marshal(arResponse)
// 	if err != nil {
// 		log.Errorf("Can't encode response: %v", err)
// 		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
// 	}
// 	log.Infof("Ready to write reponse ...")
// 	if _, err := w.Write(resp); err != nil {
// 		log.Errorf("Can't write response: %v", err)
// 		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
// 	}
// }