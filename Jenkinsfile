pipeline {

  environment {
        AGENT_IMAGE = 'quay.io/ocpmetal/assisted-installer-agent'
  }
  agent {
    node {
      label 'centos_worker'
    }
  }
  stages {
    stage('build') {
      steps {
        sh 'skipper make build'
      }
    }

    stage('test') {
       steps {
         sh 'skipper make subsystem'
       }
    }

    stage('publish images on push to master') {
        when {
            branch 'master'
        }
        steps {
            withCredentials([usernamePassword(credentialsId: 'ocpmetal_cred', passwordVariable: 'PASS', usernameVariable: 'USER')]) {
                sh '''docker login quay.io -u $USER -p $PASS'''
            }
            sh '''docker tag  ${AGENT_IMAGE} ${AGENT_IMAGE}:latest'''
            sh '''docker tag  ${AGENT_IMAGE} ${AGENT_IMAGE}:${GIT_COMMIT}'''
            sh '''docker push ${AGENT_IMAGE}:latest'''
            sh '''docker push ${AGENT_IMAGE}:${GIT_COMMIT}'''
        }

    post {
        failure {
            script {
                if (env.BRANCH_NAME == 'master')
                    stage('notify master branch fail') {
                        withCredentials([string(credentialsId: 'slack-token', variable: 'TOKEN')]) {
                            sh '''curl -X POST -H 'Content-type: application/json' --data '{"text":"Attention! master branch push integration failed, Check assisted-installer-agent"}' https://hooks.slack.com/services/${TOKEN}'''
                        }
                    }
                }
            }
        }
    }
  }
}

