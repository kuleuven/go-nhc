#!/usr/bin/env groovy

properties([
    disableConcurrentBuilds(),
])

def project = 'go-nhc'
def domain = 'gitea.icts.kuleuven.be/ceif-lnx'
def destination = 'target'
def unstash_entry_name = "${project}-stash"

node () {
    generic = new be.kuleuven.icts.Generic()
    gobuilder = new be.kuleuven.icts.Go()
    deleteDir()
    checkout scm
    generic.time(project) {
    stage(name: 'prepare dir') {
        sh "rm -rf ${destination}"
        sh "mkdir -p ${destination}/usr/bin/"
        sh "mkdir -p ${destination}/etc/bash_completion.d/"
        sh "mkdir -p ${destination}/usr/share/man/man1/"
    }
    gobuilder.build_from_dir(project: project, domain: domain, projectpath:pwd(), command: "make", go_modules: true)
    sh "mv -v ${project} ${destination}/usr/bin/"
    sh "mv -v ${project}.bash_completion ${destination}/etc/bash_completion.d/${project}"
    sh "mv -v ${project}.1.gz ${destination}/usr/share/man/man1/${project}.1.gz"

    stash name: unstash_entry_name, includes: destination+"/**/*"
    }
}

buildRpm {
    unstash = unstash_entry_name
    repository = 'icts-p-lnx-hpc-rpm-local/7'
}
