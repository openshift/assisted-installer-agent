String cron_string = BRANCH_NAME == "master" ? "@daily" : ""

pipeline {

  environment {
        AGENT_IMAGE = 'quay.io/ocpmetal/assisted-installer-agent'
        SLACK_TOKEN = credentials('slack-token')
        MASTER_SLACK_TOKEN = credentials('slack_master_token')
  }
  options {
    timeout(time: 1, unit: 'HOURS')
  }
  agent { label 'centos_worker' }
  triggers { cron(cron_string) }

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
    }
  }
  post {
        failure {
            script {
                if (env.BRANCH_NAME == 'master')
                    stage('notify master branch fail') {

                            script {
                                def data = [text: "Attention! ssisted-installer-agent branch  test failed, see: ${BUILD_URL}"]
                                writeJSON(file: 'data.txt', json: data, pretty: 4)
                            }
                            sh '''curl -X POST -H 'Content-type: application/json' --data-binary "@data.txt"  https://hooks.slack.com/services/$MASTER_SLACK_TOKEN'''
                    }
            }
        }
  }
}
