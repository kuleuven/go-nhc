['7', '8'].each {
    buildGo{
        rpm=true
        rpm_path="src/"
        rpm_params=[
            repository: "icts-p-lnx-hpc-rpm-local/${it}",
            extra_parameters: '--conflicts lbnl-nhc'
        ]
    }
}
