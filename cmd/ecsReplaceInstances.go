// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/spf13/cobra"
)

var cluster string

// ecsReplaceInstancesCmd represents the ecsReplaceInstances command
var ecsReplaceInstancesCmd = &cobra.Command{
	Use:   "ecsReplaceInstances",
	Short: "Gracefully replace EC2 instances for given ECS cluster",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		initAwsSess()

		asgName := getAsgNameForEcsCluster(cluster)
		if asgName == "" {
			fmt.Println("Unable to find ASG name for ECS cluster ", cluster)
			os.Exit(1)
		}

		instancesToTerminate := getInstanceListForAsg(asgName)

		fmt.Println("Replacing EC2 instances one at a time for ECS cluster: ", cluster)
		fmt.Println("ASG: ", asgName)

		detachAndReplaceASGInstances(asgName, instancesToTerminate)

		fmt.Printf("Terminating %v instances...\n", len(instancesToTerminate))
		for _, instanceID := range instancesToTerminate {
			_, err := terminateInstance(*instanceID)
			if err != nil {
				fmt.Println("Unable to terminate instance: ", err)
				os.Exit(1)
			}
			waitForZeroPendingTasks(cluster)
		}
		fmt.Println("Finished terminating instances")

		instances := getInstanceListForCluster(cluster)
		fmt.Println("Final instances in cluster: ", len(instances))
		fmt.Println("All done. Be sure to tip your waiter and thank AppsDev for making your life better.")
	},
}

func init() {
	RootCmd.AddCommand(ecsReplaceInstancesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ecsReplaceInstancesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ecsReplaceInstancesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	ecsReplaceInstancesCmd.Flags().StringVarP(&cluster, "cluster", "c", "", "ECS cluster name")
}

func getInstanceListForCluster(clusterName string) []string {
	svc := ecs.New(AwsSess)
	listResult, err := svc.ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(cluster),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	descResult, err := svc.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(cluster),
		ContainerInstances: listResult.ContainerInstanceArns,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	instances := []string{}

	for _, instance := range descResult.ContainerInstances {
		instances = append(instances, *instance.Ec2InstanceId)
	}

	return instances
}

func getPendingTasksCount(cluster string) int64 {
	svc := ecs.New(AwsSess)

	services, err := svc.ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(cluster),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	descResult, err := svc.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: services.ServiceArns,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}

	var pendingTasks int64

	for _, service := range descResult.Services {
		pendingTasks += *service.PendingCount
	}

	return pendingTasks
}

func terminateInstance(id string) (bool, error) {
	svc := ec2.New(AwsSess)
	instanceStatus, err := svc.DescribeInstanceStatus(&ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{&id},
	})
	if err != nil {
		return false, err
	}

	if *instanceStatus.InstanceStatuses[0].InstanceState.Name != "terminated" {
		fmt.Println("Terminating instance: ", id)
		_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
			InstanceIds: []*string{&id},
		})
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func waitForZeroPendingTasks(cluster string) {
	var pendingTasks int64

	time.Sleep(120 * time.Second)
	for pendingTasks = 1000; pendingTasks > 0; {
		time.Sleep(30 * time.Second)
		pendingTasks = getPendingTasksCount(cluster)
		fmt.Printf("\rPending tasks: %v", pendingTasks)
	}
	fmt.Println()
}

func getAsgNameForEcsCluster(cluster string) string {
	instances := getInstanceListForCluster(cluster)

	svc := ec2.New(AwsSess)
	instanceDetails, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{&instances[0]},
	})
	if err != nil {
		fmt.Println("Unable to get asg name from instance: ", err)
		os.Exit(1)
	}

	for _, tag := range instanceDetails.Reservations[0].Instances[0].Tags {
		if *tag.Key == "aws:autoscaling:groupName" {
			return *tag.Value
		}
	}

	return ""
}

func detachAndReplaceASGInstances(asgName string, instancesToTerminate []*string) {
	svc := autoscaling.New(AwsSess)

	decrement := false

	fmt.Printf("Detaching %v instances...", len(instancesToTerminate))
	_, err := svc.DetachInstances(&autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           &asgName,
		InstanceIds:                    instancesToTerminate,
		ShouldDecrementDesiredCapacity: &decrement,
	})
	if err != nil {
		fmt.Println("Unable to detach instances: ", err)
		os.Exit(1)
	}

	fmt.Printf("done\n")

	for ready := false; ready != true; {
		time.Sleep(15 * time.Second)
		instances := getInstanceListForAsg(asgName)
		fmt.Printf("\rNew instances created: %v", len(instances))
		if len(instances) == len(instancesToTerminate) {
			ready = true
			fmt.Println()
			fmt.Println("Finished creating new instances")
		}
	}
}

func getInstanceListForAsg(asgName string) []*string {
	svc := autoscaling.New(AwsSess)

	instances, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{&asgName},
	})
	if err != nil {
		fmt.Println("Unable to get list of ASG instances: ", err)
		os.Exit(1)
	}

	if len(instances.AutoScalingGroups) != 1 {
		fmt.Println("DescribeAutoScalingGroups did not return expected number of results. Expected: 1, Actual: ", len(instances.AutoScalingGroups))
		os.Exit(1)
	}

	var instanceIds []*string
	for _, ins := range instances.AutoScalingGroups[0].Instances {
		instanceIds = append(instanceIds, ins.InstanceId)
	}

	return instanceIds
}
