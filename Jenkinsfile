pipeline {
  agent any
  stages {
    stage('error') {
      agent any
      steps {
        build 'go build'
      }
    }
  }
}