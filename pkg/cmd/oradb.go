package cmd

import (
	"fmt"
	//"io"
	//"bytes"
	//"io/ioutil"
	"github.com/pkg/errors"
	//log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	//utilexec "k8s.io/client-go/util/exec"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/client-go/tools/remotecommand"
	//"k8s.io/api/extensions/v1beta1"
	//"os"
	//"os/exec"
	//"path/filepath"
	"strings"
	"time"
	"oradbauto/pkg/config"
)

type OradbOperations struct {
	configFlags      *genericclioptions.ConfigFlags
	clientset        *kubernetes.Clientset
	restConfig       *rest.Config
	rawConfig        api.Config
	oradbsts         *appsv1.StatefulSet
	oradbsvc         *corev1.Service
	oradbsvcnodeport *corev1.Service
	genericclioptions.IOStreams
	OraDbhostip            string
	UserSpecifiedCdbname   string
	UserSpecifiedPdbname   string
	UserSpecifiedNamespace string
	UserSpecifiedSyspassword   string
	UserSpecifiedCreate   bool
	UserSpecifiedDelete   bool
	UserSpecifiedList     bool
	

}

// NewOradbOperations provides an instance of OradbOperations with default values
func NewOradbOperations(streams genericclioptions.IOStreams) *OradbOperations {
	return &OradbOperations{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams: streams,
	}
}

// NewCmdOradb provides a cobra command wrapping OradbOperations
func NewCmdOradb(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewOradbOperations(streams)

	cmd := &cobra.Command{
		Use:          "kubectl-oradb list|create|delete [-c cdbname] [-p pdbname] [-w syspassword] [-n namespace]",
		Short:        "create or delete Oracle DB statefulset in OKE cluster",
		Example:      fmt.Sprintf(config.OradbExample),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			  
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(c); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&o.UserSpecifiedCdbname, "cdbname", "c", "", "User specified CDB name")
	_ = viper.BindEnv("cdbname", "KUBECTL_PLUGINS_CURRENT_CDBNAME")
	_ = viper.BindPFlag("cdbname", cmd.Flags().Lookup("cdbname"))

	cmd.Flags().StringVarP(&o.UserSpecifiedPdbname, "pdbname", "p", "", "User specified PDB name")
	_ = viper.BindEnv("pdbname", "KUBECTL_PLUGINS_CURRENT_PDBNAME")
	_ = viper.BindPFlag("pdbname", cmd.Flags().Lookup("pdbname"))

	cmd.Flags().StringVarP(&o.UserSpecifiedNamespace, "namespace", "n", "default", "User specified namespace")
	_ = viper.BindEnv("namespace", "KUBECTL_PLUGINS_CURRENT_NAMESPACE")
	_ = viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))

	cmd.Flags().StringVarP(&o.UserSpecifiedSyspassword, "syspassword", "w", "H3YX5QRE",
	"sys system password of DB")
	_ = viper.BindEnv("syspassword", "KUBECTL_PLUGINS_CURRENT_SYSPASSWORD")
	_ = viper.BindPFlag("syspassword", cmd.Flags().Lookup("syspassword"))

	return cmd
}

func (o *OradbOperations) Complete(cmd *cobra.Command, args []string) error {
	
	if len(args) != 1 {
		_ = cmd.Usage()
		return errors.New("Please check kubectl-oradb -h for usage")
	}
  
	switch strings.ToUpper(args[0]) {
	case "CREATE":
		o.UserSpecifiedCreate = true
	case "DELETE":
		o.UserSpecifiedDelete = true
	case "LIST":
		o.UserSpecifiedList = true
	default:
		_ = cmd.Usage()
		return errors.New("Please check kubectl-oradb -h for usage")
	}

	
	o.UserSpecifiedCdbname = strings.ToLower(o.UserSpecifiedCdbname) //DNS 1123 requirement
	

	var err error
	o.rawConfig, err = o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}

	o.restConfig, err = o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	o.restConfig.Timeout = 180 * time.Second
	o.clientset, err = kubernetes.NewForConfig(o.restConfig)
	if err != nil {
		return err
	}
	
	// complete db statefulset  settings
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(config.OradbStsyml), nil, nil)
	if err != nil {
        fmt.Printf("%#v", err)
		}
		
		o.oradbsts = obj.(*appsv1.StatefulSet)
		o.oradbsts.ObjectMeta.Name = o.UserSpecifiedCdbname 
		o.oradbsts.ObjectMeta.Namespace = o.UserSpecifiedNamespace
		
		//Update selector
		var oradbselector =  map[string]string {
			"oradbsts":o.UserSpecifiedCdbname + "StsSelector",
		}
		o.oradbsts.Spec.Selector.MatchLabels = oradbselector
		//Update ORACLE_SID ,ORACLE_PDB,ORACLE_PWD
		o.oradbsts.Spec.Template.Spec.Containers[0].Env[0].Value = strings.ToUpper(o.UserSpecifiedCdbname)
		o.oradbsts.Spec.Template.Spec.Containers[0].Env[1].Value = o.UserSpecifiedPdbname
		o.oradbsts.Spec.Template.Spec.Containers[0].Env[2].Value = o.UserSpecifiedSyspassword
		o.oradbsts.Spec.Template.ObjectMeta.Labels = oradbselector
    //update volume mouth and template name
		oradbvolname := o.UserSpecifiedCdbname + "-db-pv-storage"
		o.oradbsts.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name = oradbvolname
		o.oradbsts.Spec.VolumeClaimTemplates[0].ObjectMeta.Name = oradbvolname
		//fmt.Printf("%v#\n",o.oradbsts.Spec.VolumeClaimTemplates)

		//Update service name
		obj, _, err = decode([]byte(config.OradbSvcyml), nil, nil)
	  if err != nil {
        fmt.Printf("%#v", err)
		}
		o.oradbsvc = obj.(*corev1.Service)
		o.oradbsvc.ObjectMeta.Name = o.UserSpecifiedCdbname + "-svc"
		o.oradbsvc.ObjectMeta.Namespace = o.UserSpecifiedNamespace
		o.oradbsvc.Spec.Selector = oradbselector

		//Update nodeport service name
		obj, _, err = decode([]byte(config.OradbSvcymlnodeport), nil, nil)
	  if err != nil {
        fmt.Printf("%#v", err)
		}
		o.oradbsvcnodeport = obj.(*corev1.Service)
		o.oradbsvcnodeport.ObjectMeta.Name = o.UserSpecifiedCdbname + "-svc-nodeport"
		o.oradbsvcnodeport.ObjectMeta.Namespace = o.UserSpecifiedNamespace
		o.oradbsvcnodeport.Spec.Selector = oradbselector

		//find a host IP address for nodeport service connections
		NodeStatus, err := o.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	  if err != nil {
	   	panic(err)
  	} 
  	o.OraDbhostip = NodeStatus.Items[0].Status.Addresses[0].Address
    //fmt.Printf("ststatus: %v\n",o.OraDbhostip)

	return nil
}

func (o *OradbOperations) Validate(cmd *cobra.Command) error {
	//check cdb and pdb is not empty
	if o.UserSpecifiedCreate == true && (o.UserSpecifiedCdbname == "" || o.UserSpecifiedPdbname == "" ){
		_ = cmd.Usage()
		return errors.New("Please specify cdb and pdb name kubectl-oradb -h for usage")
	}

  	//check cdb is not empty
	if o.UserSpecifiedDelete == true && o.UserSpecifiedCdbname == "" {
		_ = cmd.Usage()
		return errors.New("Please specify cdb name to delete.  kubectl-oradb list for details")
	}

	if o.UserSpecifiedList {
		ListOption(o)
		return nil
	}
	return nil
}

func (o *OradbOperations) Run() error {
	
	if o.UserSpecifiedCreate {
		CreateDbOption(o)
		time.Sleep(5 * time.Second)
		CreateSvcOption(o)
		time.Sleep(3 * time.Second)
		return nil
	}
	
	if o.UserSpecifiedDelete {
		
		DeleteSvcOption(o)
		time.Sleep(3 * time.Second)
		DeleteSvcNodeportOption(o)
		time.Sleep(3 * time.Second)
		DeleteDbOption(o)
		time.Sleep(3 * time.Second)
		return nil
   }
   
return nil
}

func ListOption(o *OradbOperations) {
	
	if o.UserSpecifiedList {
		stsclient, err := o.clientset.AppsV1().StatefulSets("").List(metav1.ListOptions{
			LabelSelector: "app=peoradbauto",
			Limit:         100,
		})
				if err != nil {
						panic(err.Error())
		}
	if 	len(stsclient.Items) == 0 {
		fmt.Printf("Didn't found Oracle DB statefulset with label app=peoradbauto \n")
		return
	} else {
	for i := 0;i < len(stsclient.Items);i++ {
		fmt.Printf("Found %v statefulset with label app=peoradbauto in namespace %v\n", stsclient.Items[i].ObjectMeta.Name,stsclient.Items[i].ObjectMeta.Namespace)
		 }
	}
}
}


func DeleteDbOption(o *OradbOperations) {
	
  fmt.Printf("Deleting DB Statefulsets with label app=peoradbauto in namespace %v...\n",o.UserSpecifiedNamespace)
	Stsclient := o.clientset.AppsV1().StatefulSets(o.UserSpecifiedNamespace)
	deletePolicy := metav1.DeletePropagationForeground
	listOptions := metav1.ListOptions{
				LabelSelector: "app=peoradbauto",
				FieldSelector: "metadata.name=" + o.UserSpecifiedCdbname,
        Limit:         100,
	}
	list, err := Stsclient.List(listOptions)
	if err != nil {
		panic(err)
	}
	
	if len(list.Items) == 0 {
		fmt.Println("No statefulsets found\n")
		return
	} else {
	for _, d := range list.Items {
		fmt.Printf(" * %s \n", d.Name)
	  }
    }
    if err := Stsclient.DeleteCollection(&metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	    },listOptions); err != nil {
		panic(err)
	}
	fmt.Printf("Deleted DB statefulsets in namespace %v.\nData in PV is reserved.\n",o.UserSpecifiedNamespace)
}

func DeleteSvcOption(o *OradbOperations) {
	
  fmt.Printf("Deleting services with label app=peoradbauto in namespace %v...\n",o.UserSpecifiedNamespace)
	Svcclient := o.clientset.CoreV1().Services(o.UserSpecifiedNamespace)
	deletePolicy := metav1.DeletePropagationForeground
	listOptions := metav1.ListOptions{
				LabelSelector: "app=peoradbauto",
				FieldSelector: "metadata.name=" + o.UserSpecifiedCdbname + "-svc",
        Limit:         100,
	}
	list, err := Svcclient.List(listOptions)
	if err != nil {
		panic(err)
	}
	
	if len(list.Items) == 0 {
		fmt.Println("No Services found")
		return
	} else {
	for _, d := range list.Items {
		fmt.Printf(" * %s \n", d.Name)
	  }
    }
    if err := Svcclient.Delete(o.UserSpecifiedCdbname + "-svc", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	    }); err != nil {
		panic(err)
	}
	fmt.Printf("Deleted services in namespace %v.\n",o.UserSpecifiedNamespace)
	
}

func DeleteSvcNodeportOption(o *OradbOperations) {
	
  fmt.Printf("Deleting NodePort services with label app=peoradbauto in namespace %v...\n",o.UserSpecifiedNamespace)
	Svcclient := o.clientset.CoreV1().Services(o.UserSpecifiedNamespace)
	deletePolicy := metav1.DeletePropagationForeground
	listOptions := metav1.ListOptions{
				LabelSelector: "app=peoradbauto",
				FieldSelector: "metadata.name=" + o.UserSpecifiedCdbname + "-svc-nodeport",
        Limit:         100,
	}
	list, err := Svcclient.List(listOptions)
	if err != nil {
		panic(err)
	}
	
	if len(list.Items) == 0 {
		fmt.Println("No NodePort Services found")
		return
	} else {
	for _, d := range list.Items {
		fmt.Printf(" * %s \n", d.Name)
	  }
    }
    if err := Svcclient.Delete(o.UserSpecifiedCdbname + "-svc-nodeport", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	    }); err != nil {
		panic(err)
	}
	fmt.Printf("Deleted services in namespace %v.\n",o.UserSpecifiedNamespace)
	
}


func CreateDbOption(o *OradbOperations) {
	
	fmt.Printf("Creating Oracle CDB %v and PDB %v in namespace %v...\n",o.UserSpecifiedCdbname,o.UserSpecifiedPdbname,o.UserSpecifiedNamespace)
	Stsclient := o.clientset.AppsV1().StatefulSets(o.UserSpecifiedNamespace)
    result, err := Stsclient.Create(o.oradbsts)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created Oracle CDB %q. Use kubectl logs -f %v-0 to see alert logs\n", result.GetObjectMeta().GetName(),o.UserSpecifiedCdbname)
	
}

func CreateSvcOption(o *OradbOperations) {
	
	fmt.Printf("Creating service to serve inside K8S in namespace %v...\n",o.UserSpecifiedNamespace)
	Svcclient := o.clientset.CoreV1().Services(o.UserSpecifiedNamespace)
    result, err := Svcclient.Create(o.oradbsvc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created service %q.\n", result.GetObjectMeta().GetName())
	fmt.Printf("connect from inside K8S: system/%v@%v:1521/%v \n\n",o.UserSpecifiedSyspassword,result.GetObjectMeta().GetName(),o.UserSpecifiedPdbname)
	time.Sleep(5 * time.Second)

	fmt.Printf("Creating service to serve outside K8S in namespace %v...\n",o.UserSpecifiedNamespace)
	result, err = Svcclient.Create(o.oradbsvcnodeport)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created service %q.\n", result.GetObjectMeta().GetName())
	fmt.Printf("connect from outside K8S(ie laptop): system/%v@%v:%v/%v \n\n",o.UserSpecifiedSyspassword, o.OraDbhostip ,result.Spec.Ports[0].NodePort,o.UserSpecifiedPdbname)
	fmt.Printf("Please Wait about 8-20 min then CDB&PDB are fully up. Use kubectl logs -f %v-0 to see alert logs\n",o.UserSpecifiedCdbname)
	
}