package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	gitopsv1 "github.com/uvegla/potato/api/v1"
)

var _ = Describe("Application controller", func() {
	const (
		ApplicationName      = "test-application"
		ApplicationNamespace = "default"

		ApplicationRepository = "https://github.com/uvegla/potato-application-2"

		CowSayDeploymentName      = "cowsay"
		CowSayDeploymentNamespace = "default"
		CowSayDeploymentReplicas  = 1

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When submitting an application resource", func() {
		It("Should bring it up in the cluster", func() {
			By("By deploying all the manifests in the repository")

			ctx := context.Background()

			application := &gitopsv1.Application{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "gitops.potato.io/v1",
					Kind:       "Application",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      ApplicationName,
					Namespace: ApplicationNamespace,
				},
				Spec: gitopsv1.ApplicationSpec{
					Repository: ApplicationRepository,
					Ref:        "master",
				},
			}

			Expect(k8sClient.Create(ctx, application)).Should(Succeed())

			applicationKey := types.NamespacedName{Name: ApplicationName, Namespace: ApplicationNamespace}
			createdApplication := &gitopsv1.Application{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, applicationKey, createdApplication)

				if err != nil {
					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())

			Expect(application.Spec.Repository).Should(Equal(ApplicationRepository))

			//deploymentKey := types.NamespacedName{Name: CowSayDeploymentName, Namespace: CowSayDeploymentNamespace}
			//expectedDeployment := &appsv1.Deployment{}
			//
			//Eventually(func() bool {
			//	err := k8sClient.Get(ctx, deploymentKey, expectedDeployment)
			//
			//	if err != nil {
			//		return false
			//	}
			//
			//	return true
			//}, timeout, interval).Should(BeTrue())
			//
			//Consistently(func() (int32, error) {
			//	err := k8sClient.Get(ctx, deploymentKey, expectedDeployment)
			//
			//	if err != nil {
			//		return 0, err
			//	}
			//
			//	return *expectedDeployment.Spec.Replicas, nil
			//}, duration, interval).Should(Equal(CowSayDeploymentReplicas))
		})
	})
})
