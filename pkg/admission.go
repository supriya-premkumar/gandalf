package pkg

import (
	"strings"

	"github.com/sirupsen/logrus"
	admission "k8s.io/api/admission/v1"
	k8s "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/supriya-premkumar/gandalf/types"
)

// AdmitController implements AdmissionReviewer interface
type AdmitController struct {
	log *logrus.Entry
	cfg types.Config
}

// NewAdmissionController returns a new instance of the controller
func NewAdmissionController(logger *logrus.Logger, cfg types.Config) *AdmitController {
	return &AdmitController{
		log: logger.WithField("component", types.FixedWidthFormatter("controller")),
		cfg: cfg,
	}
}

// Review reviews the admission request and responds to the API Server with the admission result
func (a *AdmitController) Review(admissionReview *admission.AdmissionReview) (*admission.AdmissionResponse, error) {
	a.log.Infof("Reviewing admission for kind:%v | name:%v", admissionReview.Request.Kind.Kind, admissionReview.Request.Name)
	resp := &admission.AdmissionResponse{
		Allowed: false,
		Result:  &metav1.Status{},
	}

	var got map[string]string
	got = make(map[string]string)
	switch strings.ToLower(admissionReview.Request.Kind.Kind) {
	case "pod":
		pod := core.Pod{}
		deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
		if _, _, err := deserializer.Decode(admissionReview.Request.Object.Raw, nil, &pod); err != nil {
			return nil, err
		}
		got = pod.Labels

	case "deployment":
		deploy := k8s.Deployment{}
		deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
		if _, _, err := deserializer.Decode(admissionReview.Request.Object.Raw, nil, &deploy); err != nil {
			return nil, err
		}
		got = deploy.Labels

	case "replicaset":
		replica := k8s.ReplicaSet{}
		deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
		if _, _, err := deserializer.Decode(admissionReview.Request.Object.Raw, nil, &replica); err != nil {
			return nil, err
		}
		got = replica.Labels

	case "statefulset":
		sts := k8s.ReplicaSet{}
		deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
		if _, _, err := deserializer.Decode(admissionReview.Request.Object.Raw, nil, &sts); err != nil {
			return nil, err
		}
		got = sts.Labels

	case "service":
		a.log.Info("Checking for service")
		svc := core.Service{}
		deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
		if _, _, err := deserializer.Decode(admissionReview.Request.Object.Raw, nil, &svc); err != nil {
			a.log.Errorf("Failed deserialization. Err: %v", err)
			return nil, err
		}
		got = svc.Labels

	case "daemonset":
		ds := k8s.DaemonSet{}
		deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
		if _, _, err := deserializer.Decode(admissionReview.Request.Object.Raw, nil, &ds); err != nil {
			return nil, err
		}
		got = ds.Labels

	default:
		a.log.Infof("Unsupported kind %v, Cowardly refusing to enforce.", admissionReview.Request.Kind.Kind)
		resp.Allowed = true
		resp.Result.Message = "Passthrough"
		return resp, nil
	}

	a.log.Infof("Got Labels: %v", got)

	for gotKey, gotVal := range got {
		for haveKey, haveVal := range a.cfg.MatchLabels {
			if gotKey == haveKey && gotVal == haveVal {
				// Found a match, OK to admit
				a.log.Infof("OK to admit kind:%v | name:%v", admissionReview.Request.Kind.Kind, admissionReview.Request.Name)

				// Set resp.Allowed to true before returning your AdmissionResponse
				resp.Allowed = true
				return resp, nil
			}
		}
	}

	a.log.Infof("Not OK to admit kind:%v | name:%v", admissionReview.Request.Kind, admissionReview.Request.Name)

	resp.Result.Message = "admission rejected by policy"
	return resp, nil
}
