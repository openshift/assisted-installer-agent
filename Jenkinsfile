String cron_string = BRANCH_NAME == "master" ? "@daily" : ""

pipeline {
  environment {
        CI = "true"

        // Credentials
        SLACK_TOKEN = credentials('slack-token')
        QUAY_IO_CREDS = credentials('ocpmetal_cred')
        DOCKER_IO_CREDS = credentials('dockerio_cred')
  }
  options {
    timeout(time: 1, unit: 'HOURS')
  }
  agent { label 'centos_worker' }
  triggers { cron(cron_string) }

  stages {

    stage('Init') {
        steps {
            // Login to quay.io
            sh "docker login quay.io -u ${QUAY_IO_CREDS_USR} -p ${QUAY_IO_CREDS_PSW}"
            sh "podman login quay.io -u ${QUAY_IO_CREDS_USR} -p ${QUAY_IO_CREDS_PSW}"
            sh "docker login docker.io -u ${DOCKER_IO_CREDS_USR} -p ${DOCKER_IO_CREDS_PSW}"
            sh "podman login docker.io -u ${DOCKER_IO_CREDS_USR} -p ${DOCKER_IO_CREDS_PSW}"
        }
    }

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
  }
  post {
        always {
            script {
                if ((env.BRANCH_NAME == 'master') && (currentBuild.currentResult == "ABORTED" || currentBuild.currentResult == "FAILURE")){
                      script {
                          def data = [text: "Attention! ${BUILD_TAG} job failed, see: ${BUILD_URL}"]
                          writeJSON(file: 'data.txt', json: data, pretty: 4)
                      }
                      sh '''curl -X POST -H 'Content-type: application/json' --data-binary "@data.txt" https://hooks.slack.com/services/${SLACK_TOKEN}'''
                }

                sh 'sudo journalctl TAG=agent --since="1 hour ago" > agent_journalctl.log  || true'
                archiveArtifacts artifacts: '*.log', fingerprint: true
                junit '**/reports/junit*.xml'
                cobertura coberturaReportFile: '**/reports/*coverage.xml', onlyStable: false, enableNewApi: true
            }
        }
  }
}
